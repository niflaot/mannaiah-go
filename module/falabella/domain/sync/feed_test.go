package sync

import (
	"encoding/xml"
	"testing"
)

const sampleFeedXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
     <Head>
          <RequestId/>
          <RequestAction>FeedStatus</RequestAction>
          <ResponseType>FeedDetail</ResponseType>
          <Timestamp>2026-02-18T19:22:56-0300</Timestamp>
          <RequestParameters>
               <FeedID>ced2cabd-50ea-4b26-94d8-5ead4c437c45</FeedID>
          </RequestParameters>
     </Head>
     <Body>
          <FeedDetail>
               <Feed>ced2cabd-50ea-4b26-94d8-5ead4c437c45</Feed>
               <Status>Finished</Status>
               <Action>ProductCreate</Action>
               <CreationDate>2026-02-18 19:18:16</CreationDate>
               <UpdatedDate>2026-02-18 19:18:17</UpdatedDate>
               <Source>api</Source>
               <TotalRecords>1</TotalRecords>
               <ProcessedRecords>1</ProcessedRecords>
               <FailedRecords>1</FailedRecords>
               <FeedErrors>
                    <Error>
                         <Code>0</Code>
                         <Message>You have chosen an invalid brand</Message>
                         <SellerSku>7709738583245</SellerSku>
                    </Error>
                    <Error>
                         <Code>1</Code>
                         <Message>Invalid tax class</Message>
                         <SellerSku>7709738583245</SellerSku>
                    </Error>
               </FeedErrors>
          </FeedDetail>
     </Body>
</SuccessResponse>`

// TestFeedResponseUnmarshal verifies XML unmarshaling of Falabella feed status responses.
func TestFeedResponseUnmarshal(t *testing.T) {
	var response FeedResponse
	if err := xml.Unmarshal([]byte(sampleFeedXML), &response); err != nil {
		t.Fatalf("xml.Unmarshal() error = %v", err)
	}

	if response.Head.RequestAction != "FeedStatus" {
		t.Fatalf("RequestAction = %q, want %q", response.Head.RequestAction, "FeedStatus")
	}
	if response.Head.ResponseType != "FeedDetail" {
		t.Fatalf("ResponseType = %q, want %q", response.Head.ResponseType, "FeedDetail")
	}
	if response.Head.RequestParameters.FeedID != "ced2cabd-50ea-4b26-94d8-5ead4c437c45" {
		t.Fatalf("FeedID = %q, want %q", response.Head.RequestParameters.FeedID, "ced2cabd-50ea-4b26-94d8-5ead4c437c45")
	}

	detail := response.Body.FeedDetail
	if detail.Feed != "ced2cabd-50ea-4b26-94d8-5ead4c437c45" {
		t.Fatalf("Feed = %q, want %q", detail.Feed, "ced2cabd-50ea-4b26-94d8-5ead4c437c45")
	}
	if detail.Status != "Finished" {
		t.Fatalf("Status = %q, want %q", detail.Status, "Finished")
	}
	if detail.Action != "ProductCreate" {
		t.Fatalf("Action = %q, want %q", detail.Action, "ProductCreate")
	}
	if detail.TotalRecords != 1 {
		t.Fatalf("TotalRecords = %d, want %d", detail.TotalRecords, 1)
	}
	if detail.ProcessedRecords != 1 {
		t.Fatalf("ProcessedRecords = %d, want %d", detail.ProcessedRecords, 1)
	}
	if detail.FailedRecords != 1 {
		t.Fatalf("FailedRecords = %d, want %d", detail.FailedRecords, 1)
	}
	if len(detail.FeedErrors.Errors) != 2 {
		t.Fatalf("len(FeedErrors) = %d, want %d", len(detail.FeedErrors.Errors), 2)
	}

	firstError := detail.FeedErrors.Errors[0]
	if firstError.Code != 0 {
		t.Fatalf("Error[0].Code = %d, want %d", firstError.Code, 0)
	}
	if firstError.SellerSku != "7709738583245" {
		t.Fatalf("Error[0].SellerSku = %q, want %q", firstError.SellerSku, "7709738583245")
	}
}

// TestFeedDetailIsSuccess verifies feed-detail success detection behavior.
func TestFeedDetailIsSuccess(t *testing.T) {
	if detail := (FeedDetail{FailedRecords: 0}); !detail.IsSuccess() {
		t.Fatalf("FeedDetail{FailedRecords:0}.IsSuccess() = false, want true")
	}
	if detail := (FeedDetail{FailedRecords: 1}); detail.IsSuccess() {
		t.Fatalf("FeedDetail{FailedRecords:1}.IsSuccess() = true, want false")
	}
}

// TestFeedDetailIsFinished verifies feed-detail completion detection behavior.
func TestFeedDetailIsFinished(t *testing.T) {
	if detail := (FeedDetail{Status: "Finished"}); !detail.IsFinished() {
		t.Fatalf("FeedDetail{Status:Finished}.IsFinished() = false, want true")
	}
	if detail := (FeedDetail{Status: "Queued"}); detail.IsFinished() {
		t.Fatalf("FeedDetail{Status:Queued}.IsFinished() = true, want false")
	}
}

// TestFeedResponseUnmarshalEmpty verifies empty XML unmarshaling behavior.
func TestFeedResponseUnmarshalEmpty(t *testing.T) {
	var response FeedResponse
	if err := xml.Unmarshal([]byte("<SuccessResponse></SuccessResponse>"), &response); err != nil {
		t.Fatalf("xml.Unmarshal() error = %v", err)
	}

	if response.Body.FeedDetail.Feed != "" {
		t.Fatalf("Feed = %q, want empty", response.Body.FeedDetail.Feed)
	}
}
