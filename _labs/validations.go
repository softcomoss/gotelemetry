package _labs

import validation "github.com/go-ozzo/ozzo-validation/v4"

func (s Service) Validate() error {
	// Get product lines and use them for validations...
	return validation.ValidateStruct(&s,
		validation.Field(&s.ProductLine, validation.Required),
		validation.Field(&s.ServiceName, validation.Required),
		validation.Field(&s.Uptime, validation.Required),
		validation.Field(&s.BaseURL, validation.Required),
		validation.Field(&s.ServiceType, validation.Required, validation.In(Microservice, Gateway)),
		validation.Field(&s.ServiceCategory, validation.Required, validation.In(GrpcRPC, EventRPC, GraphRPC, RestRPC, APIGateway)),
	)
}
