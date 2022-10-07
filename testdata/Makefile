include .bingo/Variables.mk

buildable-v2-exists: $(BUILDABLE_V2)
	@echo "checking buildable-v2"
	@test $(BUILDABLE_V2) || (echo "no var" && exit 1)
	@test -f $(BUILDABLE_V2) || (echo "no file" && exit 1)

buildable-withreplace-exists: $(BUILDABLE_WITHREPLACE)
	@echo "checking buildable-with-replace"
	@test $(BUILDABLE_WITHREPLACE) || (echo "no var" && exit 1)
	@test -f $(BUILDABLE_WITHREPLACE) || (echo "no file" && exit 1)
