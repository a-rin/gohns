HNSSRC = hns/hns.go hns/ghns/ghns.go \
      hns/collector/collector.go hns/render/render.go \
      hns/config/config.go


all:	index.fcgi index.cgi debug server

clean:
	-rm -f index.fcgi
	-rm -f index.cgi
	go clean fcgihns.go
	go clean cgihns.go
	go clean cuihns.go


debug:	cuihns 

server: server.go $(HNSSRC)
	go build -gcflags "-N -l" server.go

index.fcgi:	fcgihns.go $(HNSSRC)
	go build -o index.fcgi fcgihns.go

index.cgi:	cgihns.go $(HNSSRC)
	go build -o index.cgi cgihns.go

cuihns:	cuihns.go $(HNSSRC)
	go build -gcflags "-N -l" cuihns.go

