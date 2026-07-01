package inbounds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/alhain1488-rgb/absolutelydisgustingpanel/backend/internal/protocols"
)

// Service holds inbound use-cases.
type Service struct {
	repo *Repo
}

// NewService builds the inbound service.
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// Repo exposes the repository.
func (s *Service) Repo() *Repo { return s.repo }

// Input carries the fields for creating/updating an inbound.
type Input struct {
	Tag      string          `json:"tag"`
	Protocol string          `json:"protocol"`
	Listen   string          `json:"listen"`
	Port     int             `json:"port"`
	Remark   string          `json:"remark"`
	Enabled  *bool           `json:"enabled"`
	Settings json.RawMessage `json:"settings_json"`
	Stream   json.RawMessage `json:"stream_settings_json"`
	Sniffing json.RawMessage `json:"sniffing_json"`
}

func (in *Input) validate() error {
	if strings.TrimSpace(in.Tag) == "" {
		return errors.New("tag is required")
	}
	if in.Port <= 0 || in.Port > 65535 {
		return errors.New("port must be 1..65535")
	}
	if _, ok := protocols.Get(in.Protocol); !ok {
		return fmt.Errorf("unknown or unsupported protocol %q", in.Protocol)
	}
	return nil
}

// Create validates, fills generated secrets and stores an inbound.
func (s *Service) Create(ctx context.Context, serverID int64, in Input) (*Inbound, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	settings, stream, err := s.prepare(in)
	if err != nil {
		return nil, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	listen := in.Listen
	if listen == "" {
		listen = "0.0.0.0"
	}
	return s.repo.Create(ctx, &Inbound{
		ServerID: serverID,
		Tag:      in.Tag,
		Protocol: in.Protocol,
		Listen:   listen,
		Port:     in.Port,
		Settings: settings,
		Stream:   stream,
		Sniffing: in.Sniffing,
		Remark:   in.Remark,
		Enabled:  enabled,
	})
}

// Update applies edits, regenerating secrets only when missing.
func (s *Service) Update(ctx context.Context, id int64, in Input) (*Inbound, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	settings, stream, err := s.prepare(in)
	if err != nil {
		return nil, err
	}
	existing.Tag = in.Tag
	existing.Protocol = in.Protocol
	if in.Listen != "" {
		existing.Listen = in.Listen
	}
	existing.Port = in.Port
	existing.Settings = settings
	existing.Stream = stream
	if len(in.Sniffing) > 0 {
		existing.Sniffing = in.Sniffing
	}
	existing.Remark = in.Remark
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}
	return s.repo.Update(ctx, existing)
}

// prepare fills protocol/transport secrets that the backend owns: REALITY keys
// and shortIds for VLESS+reality, and the method/password for Shadowsocks.
func (s *Service) prepare(in Input) (settings, stream json.RawMessage, err error) {
	set := protocols.ParseSettings(in.Settings)
	st := protocols.ParseStreamOrDefault(in.Stream)

	// REALITY key material.
	if st != nil && st.Security == "reality" {
		if st.Reality == nil {
			st.Reality = &protocols.Reality{}
		}
		if st.Reality.PrivateKey == "" || st.Reality.PublicKey == "" {
			kp, kerr := protocols.GenerateRealityKeypair()
			if kerr != nil {
				return nil, nil, kerr
			}
			st.Reality.PrivateKey = kp.PrivateKey
			st.Reality.PublicKey = kp.PublicKey
		}
		if len(st.Reality.ShortIDs) == 0 {
			sid, serr := protocols.GenerateShortID(8)
			if serr != nil {
				return nil, nil, serr
			}
			st.Reality.ShortIDs = []string{sid}
		}
		if st.Reality.Fingerprint == "" {
			st.Reality.Fingerprint = "chrome"
		}
	}

	// Shadowsocks method/password.
	if in.Protocol == "shadowsocks" {
		if set.Method == "" {
			set.Method = "aes-256-gcm"
		}
		if set.Password == "" {
			pw, perr := protocols.GenerateShadowsocksPassword()
			if perr != nil {
				return nil, nil, perr
			}
			set.Password = pw
		}
	}

	if settings, err = json.Marshal(set); err != nil {
		return nil, nil, err
	}
	if st != nil {
		if stream, err = json.Marshal(st); err != nil {
			return nil, nil, err
		}
	} else {
		stream = json.RawMessage("{}")
	}
	return settings, stream, nil
}
