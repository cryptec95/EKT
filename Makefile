all: enode ecli

enode:
	go build -o bin/enode $(PWD)/cmd/enode/main.go

ecli:
	go build -o bin/ecli $(PWD)/cmd/ecli/main.go


