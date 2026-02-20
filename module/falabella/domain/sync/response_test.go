package sync

import "testing"

const sampleProductCreateXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>90b1f884-956b-4d4e-8822-55c67fc578e0</RequestId>
    <RequestAction>ProductCreate</RequestAction>
    <ResponseType/>
    <Timestamp>2026-02-18T19:08:53-0300</Timestamp>
  </Head>
  <Body>
    <WarningDetail>
      <Field>Color</Field>
      <Message>Field 'Color' cannot be empty</Message>
      <Value>Empty</Value>
    </WarningDetail>
    <WarningDetail>
      <Field>ColorBasico</Field>
      <Message>Field 'ColorBasico' cannot be empty</Message>
      <Value>Empty</Value>
    </WarningDetail>
  </Body>
</SuccessResponse>`

const sampleProductCreateNoWarningsXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>abc-123</RequestId>
    <RequestAction>ProductCreate</RequestAction>
    <ResponseType/>
    <Timestamp>2026-02-18T19:08:53-0300</Timestamp>
  </Head>
  <Body/>
</SuccessResponse>`

const sampleProductUpdateXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>upd-456</RequestId>
    <RequestAction>ProductUpdate</RequestAction>
    <ResponseType/>
    <Timestamp>2026-02-18T20:00:00-0300</Timestamp>
  </Head>
  <Body/>
</SuccessResponse>`

// TestParseActionResponseWithWarnings verifies parsing sync action responses with warnings.
func TestParseActionResponseWithWarnings(t *testing.T) {
	response, err := ParseActionResponse([]byte(sampleProductCreateXML))
	if err != nil {
		t.Fatalf("ParseActionResponse() error = %v", err)
	}
	if response.RequestID != "90b1f884-956b-4d4e-8822-55c67fc578e0" {
		t.Fatalf("RequestID = %q, want %q", response.RequestID, "90b1f884-956b-4d4e-8822-55c67fc578e0")
	}
	if response.RequestAction != "ProductCreate" {
		t.Fatalf("RequestAction = %q, want %q", response.RequestAction, "ProductCreate")
	}
	if !response.HasWarnings() {
		t.Fatalf("HasWarnings() = false, want true")
	}
	if len(response.Warnings) != 2 {
		t.Fatalf("len(Warnings) = %d, want %d", len(response.Warnings), 2)
	}
	if response.Warnings[0].Field != "Color" {
		t.Fatalf("Warnings[0].Field = %q, want %q", response.Warnings[0].Field, "Color")
	}
	if response.Warnings[0].Message != "Field 'Color' cannot be empty" {
		t.Fatalf("Warnings[0].Message = %q, want %q", response.Warnings[0].Message, "Field 'Color' cannot be empty")
	}
	if response.Warnings[1].Field != "ColorBasico" {
		t.Fatalf("Warnings[1].Field = %q, want %q", response.Warnings[1].Field, "ColorBasico")
	}
}

// TestParseActionResponseNoWarnings verifies parsing sync action responses without warnings.
func TestParseActionResponseNoWarnings(t *testing.T) {
	response, err := ParseActionResponse([]byte(sampleProductCreateNoWarningsXML))
	if err != nil {
		t.Fatalf("ParseActionResponse() error = %v", err)
	}
	if response.RequestID != "abc-123" {
		t.Fatalf("RequestID = %q, want %q", response.RequestID, "abc-123")
	}
	if response.HasWarnings() {
		t.Fatalf("HasWarnings() = true, want false")
	}
}

// TestParseActionResponseUpdate verifies product update action detection behavior.
func TestParseActionResponseUpdate(t *testing.T) {
	response, err := ParseActionResponse([]byte(sampleProductUpdateXML))
	if err != nil {
		t.Fatalf("ParseActionResponse() error = %v", err)
	}
	if response.RequestID != "upd-456" {
		t.Fatalf("RequestID = %q, want %q", response.RequestID, "upd-456")
	}
	if response.IsCreate() {
		t.Fatalf("IsCreate() = true, want false")
	}
	if response.SyncAction() != SyncActionUpdate {
		t.Fatalf("SyncAction() = %q, want %q", response.SyncAction(), SyncActionUpdate)
	}
}

// TestActionResponseSyncAction verifies sync action mapping behavior.
func TestActionResponseSyncAction(t *testing.T) {
	createResponse := &ActionResponse{RequestAction: "ProductCreate"}
	if createResponse.SyncAction() != SyncActionCreate {
		t.Fatalf("SyncAction() = %q, want %q", createResponse.SyncAction(), SyncActionCreate)
	}
	if !createResponse.IsCreate() {
		t.Fatalf("IsCreate() = false, want true")
	}

	updateResponse := &ActionResponse{RequestAction: "ProductUpdate"}
	if updateResponse.SyncAction() != SyncActionUpdate {
		t.Fatalf("SyncAction() = %q, want %q", updateResponse.SyncAction(), SyncActionUpdate)
	}

	var nilResponse *ActionResponse
	if nilResponse.SyncAction() != SyncActionCreate {
		t.Fatalf("nil.SyncAction() = %q, want %q", nilResponse.SyncAction(), SyncActionCreate)
	}
	if nilResponse.HasWarnings() {
		t.Fatalf("nil.HasWarnings() = true, want false")
	}
}

// TestParseActionResponseEmpty verifies empty response error behavior.
func TestParseActionResponseEmpty(t *testing.T) {
	if _, err := ParseActionResponse(nil); err == nil {
		t.Fatalf("ParseActionResponse(nil) expected error")
	}
	if _, err := ParseActionResponse([]byte{}); err == nil {
		t.Fatalf("ParseActionResponse(empty) expected error")
	}
}

// TestHasRequiredFieldViolations verifies required-field violation detection behavior.
func TestHasRequiredFieldViolations(t *testing.T) {
	tests := []struct {
		name     string
		response *ActionResponse
		want     bool
	}{
		{name: "nil response", response: nil, want: false},
		{name: "no warnings", response: &ActionResponse{}, want: false},
		{name: "benign warning", response: &ActionResponse{Warnings: []Warning{
			{Field: "Foo", Message: "Some generic info"},
		}}, want: false},
		{name: "cannot be empty warning", response: &ActionResponse{Warnings: []Warning{
			{Field: "Color", Message: "Field 'Color' cannot be empty"},
		}}, want: true},
		{name: "mixed warnings with violation", response: &ActionResponse{Warnings: []Warning{
			{Field: "Foo", Message: "Just a note"},
			{Field: "ColorBasico", Message: "Field 'ColorBasico' cannot be empty"},
		}}, want: true},
		{name: "case insensitive detection", response: &ActionResponse{Warnings: []Warning{
			{Field: "Talla", Message: "CANNOT BE EMPTY"},
		}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.response.HasRequiredFieldViolations(); got != tt.want {
				t.Fatalf("HasRequiredFieldViolations() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWarningMessages verifies warning message formatting behavior.
func TestWarningMessages(t *testing.T) {
	response := &ActionResponse{
		Warnings: []Warning{
			{Field: "Color", Message: "Field 'Color' cannot be empty"},
			{Message: "Some generic warning"},
			{Field: "Talla", Message: ""},
		},
	}
	messages := response.WarningMessages()
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want %d", len(messages), 2)
	}
	if messages[0] != "[Color] Field 'Color' cannot be empty" {
		t.Fatalf("messages[0] = %q", messages[0])
	}
	if messages[1] != "Some generic warning" {
		t.Fatalf("messages[1] = %q", messages[1])
	}

	var nilResponse *ActionResponse
	if msgs := nilResponse.WarningMessages(); len(msgs) != 0 {
		t.Fatalf("nil.WarningMessages() = %#v, want nil", msgs)
	}
}
