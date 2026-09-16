# Modules whose full test suites require external services.
# Keep this list and README.md in sync when adding service-backed tests.
REDIS_TEST_MOD_DIRS := \
	./cache/redis \
	./idempotency/redis \
	./locker/redis \
	./mysql/canal/positioner/redis \
	./queue/adapter/redis \
	./ratelimit/redis

MYSQL_TEST_MOD_DIRS := ./gorm/scope
INTEGRATION_GO_MOD_DIRS := $(sort $(REDIS_TEST_MOD_DIRS) $(MYSQL_TEST_MOD_DIRS))
