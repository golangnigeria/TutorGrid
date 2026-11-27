package email

import "fmt"

func (c *Client) SendWelcomeEmail(to, firstName string) error {
	data := map[string]string{
		"UserFirstName": firstName,
	}

	return c.SendEmail(
		to,
		"Welcome to TutorGrid!",
		TemplateWelcome,
		data,
	)
}

func (c *Client) SendInvitationEmail(to, tenantName, role, inviteURL string) error {
	data := map[string]string{
		"TenantName": tenantName,
		"Role":       role,
		"InviteURL":  inviteURL,
	}

	return c.SendEmail(
		to,
		fmt.Sprintf("You're invited to join %s on TutorGrid", tenantName),
		TemplateInvitation,
		data,
	)
}
