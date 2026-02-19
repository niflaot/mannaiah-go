package sync

import "encoding/xml"

// FeedResponse defines Falabella FeedStatus XML response values.
type FeedResponse struct {
	XMLName xml.Name `xml:"SuccessResponse"`
	Head    FeedHead `xml:"Head"`
	Body    FeedBody `xml:"Body"`
}

// FeedHead defines Falabella FeedStatus response head values.
type FeedHead struct {
	// RequestID defines Falabella request identifier values.
	RequestID string `xml:"RequestId"`
	// RequestAction defines Falabella action name values.
	RequestAction string `xml:"RequestAction"`
	// ResponseType defines Falabella response type values.
	ResponseType string `xml:"ResponseType"`
	// Timestamp defines response timestamp values.
	Timestamp string `xml:"Timestamp"`
	// RequestParameters defines Falabella request parameter values.
	RequestParameters FeedRequestParameters `xml:"RequestParameters"`
}

// FeedRequestParameters defines Falabella feed request parameter values.
type FeedRequestParameters struct {
	// FeedID defines Falabella feed identifier values.
	FeedID string `xml:"FeedID"`
}

// FeedBody defines Falabella FeedStatus response body values.
type FeedBody struct {
	// FeedDetail defines feed detail values.
	FeedDetail FeedDetail `xml:"FeedDetail"`
}

// FeedDetail defines Falabella feed detail values.
type FeedDetail struct {
	// Feed defines feed identifier values.
	Feed string `xml:"Feed"`
	// Status defines feed processing status values.
	Status string `xml:"Status"`
	// Action defines feed action type values.
	Action string `xml:"Action"`
	// CreationDate defines feed creation date values.
	CreationDate string `xml:"CreationDate"`
	// UpdatedDate defines feed update date values.
	UpdatedDate string `xml:"UpdatedDate"`
	// Source defines feed request source values.
	Source string `xml:"Source"`
	// TotalRecords defines total record count values.
	TotalRecords int `xml:"TotalRecords"`
	// ProcessedRecords defines processed record count values.
	ProcessedRecords int `xml:"ProcessedRecords"`
	// FailedRecords defines failed record count values.
	FailedRecords int `xml:"FailedRecords"`
	// FeedErrors defines per-record error values.
	FeedErrors FeedErrors `xml:"FeedErrors"`
}

// FeedErrors defines Falabella feed error container values.
type FeedErrors struct {
	// Errors defines individual feed error values.
	Errors []FeedError `xml:"Error"`
}

// FeedError defines Falabella per-record feed error values.
type FeedError struct {
	// Code defines error code values.
	Code int `xml:"Code"`
	// Message defines error message values.
	Message string `xml:"Message"`
	// SellerSku defines affected seller SKU values.
	SellerSku string `xml:"SellerSku"`
}

// IsSuccess reports whether the feed completed without failed records.
func (d FeedDetail) IsSuccess() bool {
	return d.FailedRecords == 0
}

// IsFinished reports whether the feed has completed processing.
func (d FeedDetail) IsFinished() bool {
	return d.Status == "Finished"
}
