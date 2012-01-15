include $(GOROOT)/src/Make.inc

TARG=urlwatch
GOFILES=main.go notifiers.go

include $(GOROOT)/src/Make.cmd
