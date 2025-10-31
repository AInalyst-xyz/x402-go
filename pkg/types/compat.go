package types

// AlternativeVerifyRequest supports the alternative JSON format with camelCase and different structure
type AlternativeVerifyRequest struct {
	X402Version         int                            `json:"x402Version"`
	PaymentPayload      AlternativePaymentPayload      `json:"paymentPayload"`
	PaymentRequirements AlternativePaymentRequirements `json:"paymentRequirements"`
}

// AlternativePaymentPayload represents the payment payload in alternative format
type AlternativePaymentPayload struct {
	X402Version int                   `json:"x402Version"`
	Scheme      string                `json:"scheme"`
	Network     string                `json:"network"`
	Payload     AlternativeEvmPayload `json:"payload"`
}

// AlternativeEvmPayload represents the EVM payload in alternative format
type AlternativeEvmPayload struct {
	Signature     string                             `json:"signature"`
	Authorization AlternativeEvmPayloadAuthorization `json:"authorization"`
}

// AlternativeEvmPayloadAuthorization represents authorization in alternative format
type AlternativeEvmPayloadAuthorization struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	ValidAfter  string `json:"validAfter"`  // String instead of uint64
	ValidBefore string `json:"validBefore"` // String instead of uint64
	Nonce       string `json:"nonce"`
}

// AlternativePaymentRequirements represents payment requirements in alternative format
type AlternativePaymentRequirements struct {
	Scheme            string                 `json:"scheme"`
	Network           string                 `json:"network"`
	MaxAmountRequired string                 `json:"maxAmountRequired"`
	Resource          string                 `json:"resource,omitempty"`
	Description       string                 `json:"description,omitempty"`
	MimeType          string                 `json:"mimeType,omitempty"`
	PayTo             string                 `json:"payTo"`
	MaxTimeoutSeconds int                    `json:"maxTimeoutSeconds,omitempty"`
	Asset             string                 `json:"asset"`
	OutputSchema      map[string]interface{} `json:"outputSchema,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
}

// // ToStandardFormat converts AlternativeVerifyRequest to standard VerifyRequest
// func (a *AlternativeVerifyRequest) ToStandardFormat() (*VerifyRequest, error) {
// 	// Convert validAfter and validBefore from string to uint64

// 	// Build standard format
// 	standard := &VerifyRequest{
// 		PaymentPayload: PaymentPayload{
// 			X402Version: a.PaymentPayload.X402Version,
// 			Scheme:      Scheme(a.PaymentPayload.Scheme),
// 			Network:     Network(a.PaymentPayload.Network),
// 			Payload: ExactEvmPayload{
// 				Signature: a.PaymentPayload.Payload.Signature,
// 				Authorization: ExactEvmPayloadAuthorization{
// 					From:        common.HexToAddress(a.PaymentPayload.Payload.Authorization.From),
// 					To:          common.HexToAddress(a.PaymentPayload.Payload.Authorization.To),
// 					Value:       a.PaymentPayload.Payload.Authorization.Value,
// 					ValidAfter:  a.PaymentPayload.Payload.Authorization.ValidAfter,
// 					ValidBefore: a.PaymentPayload.Payload.Authorization.ValidBefore,
// 					Nonce:       a.PaymentPayload.Payload.Authorization.Nonce,
// 				},
// 			},
// 		},
// 		PaymentRequirements: PaymentRequirements{
// 			Version: X402VersionV1,
// 			Scheme:  Scheme(a.PaymentRequirements.Scheme),
// 			Network: Network(a.PaymentRequirements.Network),
// 			PayTo: MixedAddress{
// 				Type:    "evm",
// 				Address: a.PaymentRequirements.PayTo,
// 			},
// 			MaxAmountRequired: a.PaymentRequirements.MaxAmountRequired,
// 			Resource:          a.PaymentRequirements.Resource,
// 			Description:       a.PaymentRequirements.Description,
// 			MimeType:          a.PaymentRequirements.MimeType,
// 			MaxTimeoutSeconds: a.PaymentRequirements.MaxTimeoutSeconds,
// 			Asset: MixedAddress{
// 				Type:    "evm",
// 				Address: a.PaymentRequirements.Asset,
// 			},
// 			Extra: json.RawMessage(extraBytes),
// 		},
// 	}

// 	return standard, nil
// }

// // UnmarshalVerifyRequest attempts to unmarshal from either standard or alternative format
// func UnmarshalVerifyRequest(data []byte) (*VerifyRequest, error) {
// 	// Try standard format first
// 	var standard VerifyRequest
// 	if err := json.Unmarshal(data, &standard); err == nil {
// 		// Check if it's actually standard format (has payment_payload not paymentPayload)
// 		var raw map[string]interface{}
// 		json.Unmarshal(data, &raw)
// 		if _, ok := raw["payment_payload"]; ok {
// 			return &standard, nil
// 		}
// 	}

// 	// Try alternative format
// 	var alternative AlternativeVerifyRequest
// 	if err := json.Unmarshal(data, &alternative); err != nil {
// 		return nil, fmt.Errorf("failed to parse request in either format: %w", err)
// 	}

// 	// Convert to standard format
// 	return alternative.ToStandardFormat()
// }
