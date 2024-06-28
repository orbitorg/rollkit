package namespace

import tmrand "github.com/rollkit/rollkit/libs/rand"

func RandomNamespace() Namespace {
	for {
		id := RandomVerzionZeroID()
		namespace, err := New(NamespaceVersionZero, id)
		if err != nil {
			continue
		}
		return namespace
	}
}

func RandomVerzionZeroID() []byte {
	return append(NamespaceVersionZeroPrefix, tmrand.Bytes(NamespaceVersionZeroIDSize)...)
}
