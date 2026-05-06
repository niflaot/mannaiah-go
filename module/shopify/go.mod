module mannaiah/module/shopify

go 1.25.5

require (
	github.com/getkin/kin-openapi v0.133.0
	go.uber.org/zap v1.27.1
	gorm.io/gorm v1.31.1
	mannaiah/module/contacts v0.0.0
	mannaiah/module/core v0.0.0
	mannaiah/module/orders v0.0.0
)

replace mannaiah/module/core => ../core

replace mannaiah/module/contacts => ../contacts

replace mannaiah/module/orders => ../orders
