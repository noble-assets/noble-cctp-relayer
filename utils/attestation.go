package utils

// Attestation is the hex encoded attestation received from Circle
// Message is the message bytes
// Key is the hex encoded keccack256 hash of the message bytes
type Attestation struct {
	Attestation string
	Message     []byte
	Key         string
}

// AttestationResponse is the response received from Circle's iris api
// Example: https://iris-api-sandbox.circle.com/attestations/0x85bbf7e65a5992e6317a61f005e06d9972a033d71b514be183b179e1b47723fe
type AttestationResponse struct {
	Attestation string
	Status      string
}
