package main

import (
	"context"
	"errors"
	"testing"

	contactapplication "mannaiah/module/contacts/application"
	contactdomain "mannaiah/module/contacts/domain"
	contactport "mannaiah/module/contacts/port"
)

type mockCampaignContactService struct {
	getFn func(ctx context.Context, id string) (*contactdomain.Contact, error)
}

// Create is a no-op test stub.
func (m mockCampaignContactService) Create(_ context.Context, _ contactapplication.CreateCommand) (*contactdomain.Contact, error) {
	return nil, errors.New("not implemented")
}

// Get resolves one contact for test scenarios.
func (m mockCampaignContactService) Get(ctx context.Context, id string) (*contactdomain.Contact, error) {
	if m.getFn == nil {
		return nil, contactapplication.ErrInvalidID
	}

	return m.getFn(ctx, id)
}

// List is a no-op test stub.
func (m mockCampaignContactService) List(_ context.Context, _ contactport.ListQuery) (*contactapplication.ListResult, error) {
	return nil, errors.New("not implemented")
}

// Update is a no-op test stub.
func (m mockCampaignContactService) Update(_ context.Context, _ string, _ contactapplication.UpdateCommand) (*contactdomain.Contact, error) {
	return nil, errors.New("not implemented")
}

// Delete is a no-op test stub.
func (m mockCampaignContactService) Delete(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

// TestCampaignContactDataProviderAdapter_GetContactData verifies contact names/emails are resolved from contacts service.
func TestCampaignContactDataProviderAdapter_GetContactData(t *testing.T) {
	t.Parallel()

	adapter := campaignContactDataProviderAdapter{
		contacts: mockCampaignContactService{
			getFn: func(_ context.Context, id string) (*contactdomain.Contact, error) {
				if id != "83395cf06d6837104f19a7c9a99a2517" {
					t.Fatalf("contact id = %q, want expected id", id)
				}

				return &contactdomain.Contact{
					LegalName: "Juliana Marcela Villegas Sarmiento",
					Email:     "julianamvillegassarmiento@gmail.com",
				}, nil
			},
		},
	}

	data, err := adapter.GetContactData(context.Background(), "83395cf06d6837104f19a7c9a99a2517")
	if err != nil {
		t.Fatalf("GetContactData() error = %v", err)
	}
	if data.Name != "Juliana Marcela Villegas Sarmiento" {
		t.Fatalf("name = %q, want legal name", data.Name)
	}
	if data.Email != "julianamvillegassarmiento@gmail.com" {
		t.Fatalf("email = %q, want contact email", data.Email)
	}
}

// TestResolveContactDisplayName verifies legal-name and personal-name fallback behavior.
func TestResolveContactDisplayName(t *testing.T) {
	t.Parallel()

	withLegal := resolveContactDisplayName(&contactdomain.Contact{
		LegalName: "Empresa SAS",
		FirstName: "Ana",
		LastName:  "Perez",
	})
	if withLegal != "Empresa SAS" {
		t.Fatalf("with legal = %q, want legal name", withLegal)
	}

	withPersonal := resolveContactDisplayName(&contactdomain.Contact{
		FirstName: "Juliana",
		LastName:  "Villegas",
	})
	if withPersonal != "Juliana Villegas" {
		t.Fatalf("with personal = %q, want full personal name", withPersonal)
	}
}
