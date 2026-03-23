package store

import "time"

// shippingMarkModel defines shipping mark persistence row values.
type shippingMarkModel struct {
	ID                    string                  `gorm:"column:id;type:varchar(64);primaryKey"`
	OrderID               string                  `gorm:"column:order_id;type:varchar(255);index"`
	CarrierID             string                  `gorm:"column:carrier_id;type:varchar(100);index"`
	TrackingNumber        *string                 `gorm:"column:tracking_number;type:varchar(255);uniqueIndex"`
	Status                string                  `gorm:"column:status;type:varchar(50)"`
	DocumentType          *string                 `gorm:"column:document_type;type:varchar(10)"`
	DocumentRef           *string                 `gorm:"column:document_ref;type:text"`
	SenderName            string                  `gorm:"column:sender_name;type:varchar(255)"`
	SenderID              string                  `gorm:"column:sender_id;type:varchar(50)"`
	SenderIDType          string                  `gorm:"column:sender_id_type;type:varchar(10)"`
	SenderAddress         string                  `gorm:"column:sender_address;type:varchar(500)"`
	SenderCityCode        string                  `gorm:"column:sender_city_code;type:varchar(20)"`
	SenderPhone           string                  `gorm:"column:sender_phone;type:varchar(50)"`
	SenderEmail           string                  `gorm:"column:sender_email;type:varchar(255)"`
	RecipientName         string                  `gorm:"column:recipient_name;type:varchar(255)"`
	RecipientID           string                  `gorm:"column:recipient_id;type:varchar(50)"`
	RecipientIDType       string                  `gorm:"column:recipient_id_type;type:varchar(10)"`
	RecipientAddress      string                  `gorm:"column:recipient_address;type:varchar(500)"`
	RecipientCityCode     string                  `gorm:"column:recipient_city_code;type:varchar(20)"`
	RecipientPhone        string                  `gorm:"column:recipient_phone;type:varchar(50)"`
	RecipientEmail        string                  `gorm:"column:recipient_email;type:varchar(255)"`
	TotalWeight           float64                 `gorm:"column:total_weight;type:decimal(10,2)"`
	TotalVolumetricWeight float64                 `gorm:"column:total_volumetric_weight;type:decimal(10,2)"`
	DeclaredValue         float64                 `gorm:"column:declared_value;type:decimal(15,2)"`
	PaymentForm           string                  `gorm:"column:payment_form;type:varchar(50)"`
	Observations          string                  `gorm:"column:observations;type:text"`
	DispatchBatchID       *string                 `gorm:"column:dispatch_batch_id;type:varchar(64);index"`
	CreatedAt             time.Time               `gorm:"column:created_at"`
	UpdatedAt             time.Time               `gorm:"column:updated_at"`
	Units                 []shippingMarkUnitModel `gorm:"foreignKey:ShippingMarkID;references:ID"`
}

// TableName defines shipping mark table names.
func (shippingMarkModel) TableName() string {
	return "shipping_marks"
}

// shippingMarkUnitModel defines shipping mark unit row values.
type shippingMarkUnitModel struct {
	ID                 string  `gorm:"column:id;type:varchar(64);primaryKey"`
	ShippingMarkID     string  `gorm:"column:shipping_mark_id;type:varchar(64);index"`
	Description        string  `gorm:"column:description;type:varchar(500)"`
	PackageType        string  `gorm:"column:package_type;type:varchar(50)"`
	HeightCM           float64 `gorm:"column:height_cm;type:decimal(8,2)"`
	WidthCM            float64 `gorm:"column:width_cm;type:decimal(8,2)"`
	DepthCM            float64 `gorm:"column:depth_cm;type:decimal(8,2)"`
	RealWeightKG       float64 `gorm:"column:real_weight_kg;type:decimal(8,2)"`
	VolumetricWeightKG float64 `gorm:"column:volumetric_weight_kg;type:decimal(8,2)"`
	DeclaredValue      float64 `gorm:"column:declared_value;type:decimal(15,2)"`
}

// TableName defines shipping mark unit table names.
func (shippingMarkUnitModel) TableName() string {
	return "shipping_mark_units"
}

// dispatchBatchModel defines dispatch batch row values.
type dispatchBatchModel struct {
	ID        string     `gorm:"column:id;type:varchar(64);primaryKey"`
	Name      string     `gorm:"column:name;type:varchar(255)"`
	CarrierID string     `gorm:"column:carrier_id;type:varchar(100);index"`
	Status    string     `gorm:"column:status;type:varchar(20);index"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	ClosedAt  *time.Time `gorm:"column:closed_at"`
}

// TableName defines dispatch batch table names.
func (dispatchBatchModel) TableName() string {
	return "dispatch_batches"
}

// quotationModel defines quotation persistence row values.
type quotationModel struct {
	ID              string    `gorm:"column:id;type:varchar(64);primaryKey"`
	OrderID         string    `gorm:"column:order_id;type:varchar(255);index"`
	CarrierID       string    `gorm:"column:carrier_id;type:varchar(100);index"`
	OriginCityCode  string    `gorm:"column:origin_city_code;type:varchar(20)"`
	DestCityCode    string    `gorm:"column:dest_city_code;type:varchar(20)"`
	FreightCost     float64   `gorm:"column:freight_cost;type:decimal(15,2)"`
	EstimatedDays   int       `gorm:"column:estimated_days"`
	CurrencyCode    string    `gorm:"column:currency_code;type:varchar(5)"`
	ExpiresAt       time.Time `gorm:"column:expires_at"`
	RequestSnapshot string    `gorm:"column:request_snapshot;type:text"`
	RawResponse     string    `gorm:"column:raw_response;type:text"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

// TableName defines quotation table names.
func (quotationModel) TableName() string {
	return "shipping_quotations"
}
