package servers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/crypto"
	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/ssh"
)

// RunnerFactory builds an ssh.Runner for a server config. Tests inject a fake.
type RunnerFactory func(cfg ssh.Config) ssh.Runner

// defaultRunnerFactory returns a real SSH client.
func defaultRunnerFactory(cfg ssh.Config) ssh.Runner { return ssh.NewClient(cfg) }

// Service holds the server use-cases.
type Service struct {
	repo    *Repo
	cipher  *crypto.Cipher
	geo     Geolocator
	factory RunnerFactory
	resolve func(host string) ([]string, error)
}

// NewService builds the server service. geo may be nil to skip geolocation.
func NewService(repo *Repo, cipher *crypto.Cipher, geo Geolocator) *Service {
	return &Service{
		repo:    repo,
		cipher:  cipher,
		geo:     geo,
		factory: defaultRunnerFactory,
		resolve: net.LookupHost,
	}
}

// SetRunnerFactory overrides how SSH runners are built (for tests).
func (s *Service) SetRunnerFactory(f RunnerFactory) { s.factory = f }

// SetResolver overrides DNS resolution (for tests).
func (s *Service) SetResolver(f func(string) ([]string, error)) { s.resolve = f }

// Repo exposes the repository.
func (s *Service) Repo() *Repo { return s.repo }

// CreateInput carries the fields for creating a server, with plaintext SSH secret.
type CreateInput struct {
	Name            string
	Host            string
	SSHPort         int
	SSHUser         string
	SSHAuthMethod   string
	SSHSecret       string // plaintext, encrypted before storage
	SSHPassphrase   string // plaintext, encrypted before storage
	XrayConfigPath  string
	XrayServiceName string
}

func (in *CreateInput) applyDefaults() {
	if in.SSHPort == 0 {
		in.SSHPort = 22
	}
	if in.XrayConfigPath == "" {
		in.XrayConfigPath = "/usr/local/etc/xray/config.json"
	}
	if in.XrayServiceName == "" {
		in.XrayServiceName = "xray"
	}
}

func (in *CreateInput) validate() error {
	if strings.TrimSpace(in.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(in.Host) == "" {
		return errors.New("host is required")
	}
	if strings.TrimSpace(in.SSHUser) == "" {
		return errors.New("ssh_user is required")
	}
	if in.SSHAuthMethod != string(ssh.AuthKey) && in.SSHAuthMethod != string(ssh.AuthPassword) {
		return fmt.Errorf("ssh_auth_method must be 'key' or 'password'")
	}
	return nil
}

// Create validates, encrypts the SSH secret and stores the server.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Server, error) {
	in.applyDefaults()
	if err := in.validate(); err != nil {
		return nil, err
	}
	secretEnc, err := s.encrypt(in.SSHSecret)
	if err != nil {
		return nil, err
	}
	passEnc, err := s.encrypt(in.SSHPassphrase)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, &Server{
		Name:            in.Name,
		Host:            in.Host,
		SSHPort:         in.SSHPort,
		SSHUser:         in.SSHUser,
		SSHAuthMethod:   in.SSHAuthMethod,
		SSHSecretEnc:    secretEnc,
		SSHPassphrase:   passEnc,
		XrayConfigPath:  in.XrayConfigPath,
		XrayServiceName: in.XrayServiceName,
	})
}

// Update applies edits. A non-empty SSHSecret replaces the stored secret.
func (s *Service) Update(ctx context.Context, id int64, in CreateInput) (*Server, error) {
	in.applyDefaults()
	if err := in.validate(); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	existing.Name = in.Name
	existing.Host = in.Host
	existing.SSHPort = in.SSHPort
	existing.SSHUser = in.SSHUser
	existing.SSHAuthMethod = in.SSHAuthMethod
	existing.XrayConfigPath = in.XrayConfigPath
	existing.XrayServiceName = in.XrayServiceName
	// Only re-encrypt if a new secret/passphrase was provided.
	existing.SSHSecretEnc = ""
	existing.SSHPassphrase = ""
	if in.SSHSecret != "" {
		if existing.SSHSecretEnc, err = s.encrypt(in.SSHSecret); err != nil {
			return nil, err
		}
	}
	if in.SSHPassphrase != "" {
		if existing.SSHPassphrase, err = s.encrypt(in.SSHPassphrase); err != nil {
			return nil, err
		}
	}
	return s.repo.Update(ctx, existing)
}

// runnerFor decrypts the server's SSH secret and builds a runner.
func (s *Service) runnerFor(srv *Server) (ssh.Runner, error) {
	secret, err := s.decrypt(srv.SSHSecretEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt ssh secret: %w", err)
	}
	pass, err := s.decrypt(srv.SSHPassphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypt ssh passphrase: %w", err)
	}
	return s.factory(ssh.Config{
		Host:       srv.Host,
		Port:       srv.SSHPort,
		User:       srv.SSHUser,
		AuthMethod: ssh.AuthMethod(srv.SSHAuthMethod),
		Secret:     secret,
		Passphrase: pass,
	}), nil
}

// RunnerFor is exported for other packages (e.g. sync) that need a runner.
func (s *Service) RunnerFor(srv *Server) (ssh.Runner, error) { return s.runnerFor(srv) }

// Check pings the server, detects Xray status, resolves IP/geo and persists.
func (s *Service) Check(ctx context.Context, id int64) (*Server, error) {
	srv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	runner, err := s.runnerFor(srv)
	if err != nil {
		return nil, err
	}

	status := StatusOnline
	if err := ssh.Ping(ctx, runner); err != nil {
		// Unreachable: persist offline and return the updated record.
		_ = s.repo.UpdateCheckResult(ctx, id, StatusOffline, srv.IP, srv.GeoCountry, srv.GeoCity, srv.GeoASN)
		return s.repo.GetByID(ctx, id)
	}

	// Xray status folds into the overall status only as extra info; reachable
	// hosts are "online". (Xray-specific state is surfaced via stats/logs.)
	if _, derr := ssh.DetectXray(ctx, runner, srv.XrayServiceName); derr != nil {
		status = StatusOnline
	}

	ip := s.resolveIP(srv.Host)
	country, city, asn := srv.GeoCountry, srv.GeoCity, srv.GeoASN
	if s.geo != nil && ip != "" {
		if g, gerr := s.geo.Lookup(ctx, ip); gerr == nil {
			country, city, asn = g.Country, g.City, g.ASN
		}
	}

	if err := s.repo.UpdateCheckResult(ctx, id, status, ip, country, city, asn); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

// Stats collects resource metrics from the server.
func (s *Service) Stats(ctx context.Context, id int64) (*ssh.Stats, error) {
	srv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	runner, err := s.runnerFor(srv)
	if err != nil {
		return nil, err
	}
	return ssh.CollectStats(ctx, runner)
}

// RestartXray restarts the Xray service on the server.
func (s *Service) RestartXray(ctx context.Context, id int64) error {
	srv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	runner, err := s.runnerFor(srv)
	if err != nil {
		return err
	}
	return ssh.RestartXray(ctx, runner, srv.XrayServiceName)
}

func (s *Service) resolveIP(host string) string {
	if net.ParseIP(host) != nil {
		return host
	}
	if s.resolve == nil {
		return ""
	}
	addrs, err := s.resolve(host)
	if err != nil || len(addrs) == 0 {
		return ""
	}
	return addrs[0]
}

func (s *Service) encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	return s.cipher.EncryptString(plain)
}

func (s *Service) decrypt(enc string) (string, error) {
	if enc == "" {
		return "", nil
	}
	return s.cipher.DecryptString(enc)
}
