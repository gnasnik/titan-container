package cert

import "encoding/asn1"

var AuthVersionOID = asn1.ObjectIdentifier{2, 23, 133, 2, 6}

const (
	PemBlkTypeCertificate = "CERTIFICATE"
)
