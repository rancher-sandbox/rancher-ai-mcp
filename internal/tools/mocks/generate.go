//go:generate go tool -modfile ../../../gotools/mockgen/go.mod mockgen -source=../tools.go -destination=./tools.go -package=mocks
//go:generate go tool -modfile ../../../gotools/mockgen/go.mod mockgen -destination=./dynamic.go -package=mocks k8s.io/client-go/dynamic ResourceInterface

package mocks
