package protocols

import "fmt"

// clientEmail builds a stable per-client identifier used as xray's "email"
// (its user tag). Combines the inbound tag and client to stay unique.
func clientEmail(tag string, c Client) string {
	name := c.Name
	if name == "" {
		name = c.UUID
	}
	return fmt.Sprintf("%s-%s", tag, name)
}

// linkRemark is the human-facing label shown in the client app (URI fragment).
func linkRemark(in Inbound, c Client) string {
	label := in.Remark
	if label == "" {
		label = in.Tag
	}
	if c.Name != "" {
		return fmt.Sprintf("%s-%s", label, c.Name)
	}
	return label
}
