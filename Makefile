PORT = 8080
TIMEZONE = America/Sao_Paulo
PERIODS = '{"breakfast":[0,10],"lunch":[10,15],"dinner":[15,0]}'
PEOPLE = '{"alice":"alice","bob":"bob"}'
ENTRIES = '{ \
	"Trendy Neighborhood": { \
		"Morning Bakery": {"cost":2,"open":{"tue":["breakfast","lunch"],"wed":["breakfast","lunch"],"thu":["breakfast","lunch"],"fri":["breakfast","lunch"],"sat":["breakfast","lunch"],"sun":["breakfast","lunch"]}}, \
		"Corner Cafe": {"cost":2,"open":{"mon":["breakfast","lunch"],"tue":["breakfast","lunch"],"wed":["breakfast","lunch"],"thu":["breakfast","lunch"],"fri":["breakfast","lunch"],"sat":["breakfast"],"sun":["breakfast"]}}, \
		"Neighborhood Bistro": {"cost":2,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
		"Family Grill": {"cost":2,"open":{"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch","dinner"]}}, \
		"Soup and Salad Bar": {"cost":2,"open":{"mon":["lunch"],"tue":["lunch"],"wed":["lunch"],"thu":["lunch"],"fri":["lunch"]}}, \
		"Cozy Pasta House": {"cost":2,"open":{"mon":["dinner"],"tue":["dinner"],"wed":["dinner"],"thu":["dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
		"Seaside Seafood": {"cost":3,"open":{"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}} \
	}, \
	"Downtown": { \
		"Burger Joint": {"cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch","dinner"]}}, \
		"Greasy Fast Food": {"cost":1,"open":{"mon":["breakfast","lunch","dinner"],"tue":["breakfast","lunch","dinner"],"wed":["breakfast","lunch","dinner"],"thu":["breakfast","lunch","dinner"],"fri":["breakfast","lunch","dinner"],"sat":["breakfast","lunch","dinner"],"sun":["breakfast","lunch","dinner"]}}, \
		"Quick Wrap": {"cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch"]}}, \
		"Fried Chicken Shack": {"cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch","dinner"]}}, \
		"Hot Dog Cart": {"cost":1,"open":{"mon":["lunch"],"tue":["lunch"],"wed":["lunch"],"thu":["lunch"],"fri":["lunch","dinner"]}}, \
		"Pizza Corner": {"cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"]}}, \
		"Noodle Express": {"cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"]}} \
	}, \
	"Upscale Street": { \
		"Fancy Steakhouse": {"cost":4,"open":{"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
		"Chef Table": {"cost":4,"open":{"thu":["dinner"],"fri":["dinner"],"sat":["dinner"]}}, \
		"French Cuisine": {"cost":4,"open":{"wed":["dinner"],"thu":["dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"]}}, \
		"Elegant Sushi": {"cost":4,"open":{"tue":["dinner"],"wed":["dinner"],"thu":["dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
		"Rooftop Lounge": {"cost":3,"open":{"thu":["dinner"],"fri":["dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
		"Wine and Tapas": {"cost":3,"open":{"wed":["dinner"],"thu":["dinner"],"fri":["dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}} \
	} \
}'

.PHONY: dev test

dev:
	PORT=$(PORT) \
	TIMEZONE=$(TIMEZONE) \
	PERIODS=$(PERIODS) \
	PEOPLE=$(PEOPLE) \
	ENTRIES=$(ENTRIES) \
	go run -buildvcs=true ./cmd/anythingsrv

test:
	go test ./... -count=1 -cover -coverprofile /tmp/cover.out
