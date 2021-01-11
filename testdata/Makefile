include .bingo/Variables.mk

faillint-exists: $(FAILLINT)
	@echo "checking faillint"
	@test $(FAILLINT) || (echo "no var" && exit 1)
	@test -f $(FAILLINT) || (echo "no file" && exit 1)

buildable-exists: $(BUILDABLE)
	@echo "checking buildable"
	@test $(BUILDABLE) || (echo "no var" && exit 1)
	@test -f $(BUILDABLE) || (echo "no file" && exit 1)

wr_buildable-exists: $(WR_BUILDABLE)
	@echo "checking wr_buildable"
	@test $(WR_BUILDABLE) || (echo "no var" && exit 1)
	@test -f $(WR_BUILDABLE) || (echo "no file" && exit 1)
