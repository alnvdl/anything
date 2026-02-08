PORT = 8080
TIMEZONE = America/Sao_Paulo
PERIODS = '{"breakfast":[0,10],"lunch":[10,15],"dinner":[15,0]}'
PEOPLE = '{"alice":"token-alice","bob":"token-bob"}'
ENTRIES = '[ \
	{"name":"Burger Joint","group":"Downtown","cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch","dinner"]}}, \
	{"name":"Greasy Fast Food","group":"Downtown","cost":1,"open":{"mon":["breakfast","lunch","dinner"],"tue":["breakfast","lunch","dinner"],"wed":["breakfast","lunch","dinner"],"thu":["breakfast","lunch","dinner"],"fri":["breakfast","lunch","dinner"],"sat":["breakfast","lunch","dinner"],"sun":["breakfast","lunch","dinner"]}}, \
	{"name":"Quick Wrap","group":"Downtown","cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch"]}}, \
	{"name":"Fried Chicken Shack","group":"Downtown","cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch","dinner"]}}, \
	{"name":"Hot Dog Cart","group":"Downtown","cost":1,"open":{"mon":["lunch"],"tue":["lunch"],"wed":["lunch"],"thu":["lunch"],"fri":["lunch","dinner"]}}, \
	{"name":"Pizza Corner","group":"Downtown","cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"]}}, \
	{"name":"Noodle Express","group":"Downtown","cost":1,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"]}}, \
	{"name":"Morning Bakery","group":"Trendy Neighborhood","cost":2,"open":{"tue":["breakfast","lunch"],"wed":["breakfast","lunch"],"thu":["breakfast","lunch"],"fri":["breakfast","lunch"],"sat":["breakfast","lunch"],"sun":["breakfast","lunch"]}}, \
	{"name":"Corner Cafe","group":"Trendy Neighborhood","cost":2,"open":{"mon":["breakfast","lunch"],"tue":["breakfast","lunch"],"wed":["breakfast","lunch"],"thu":["breakfast","lunch"],"fri":["breakfast","lunch"],"sat":["breakfast"],"sun":["breakfast"]}}, \
	{"name":"Neighborhood Bistro","group":"Trendy Neighborhood","cost":2,"open":{"mon":["lunch","dinner"],"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
	{"name":"Family Grill","group":"Trendy Neighborhood","cost":2,"open":{"tue":["lunch","dinner"],"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch","dinner"]}}, \
	{"name":"Soup and Salad Bar","group":"Trendy Neighborhood","cost":2,"open":{"mon":["lunch"],"tue":["lunch"],"wed":["lunch"],"thu":["lunch"],"fri":["lunch"]}}, \
	{"name":"Cozy Pasta House","group":"Trendy Neighborhood","cost":2,"open":{"mon":["dinner"],"tue":["dinner"],"wed":["dinner"],"thu":["dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
	{"name":"Seaside Seafood","group":"Trendy Neighborhood","cost":3,"open":{"wed":["lunch","dinner"],"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
	{"name":"Fancy Steakhouse","group":"Upscale Street","cost":4,"open":{"thu":["lunch","dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
	{"name":"Chef Table","group":"Upscale Street","cost":4,"open":{"thu":["dinner"],"fri":["dinner"],"sat":["dinner"]}}, \
	{"name":"French Cuisine","group":"Upscale Street","cost":4,"open":{"wed":["dinner"],"thu":["dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"]}}, \
	{"name":"Elegant Sushi","group":"Upscale Street","cost":4,"open":{"tue":["dinner"],"wed":["dinner"],"thu":["dinner"],"fri":["lunch","dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
	{"name":"Rooftop Lounge","group":"Upscale Street","cost":3,"open":{"thu":["dinner"],"fri":["dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}}, \
	{"name":"Wine and Tapas","group":"Upscale Street","cost":3,"open":{"wed":["dinner"],"thu":["dinner"],"fri":["dinner"],"sat":["lunch","dinner"],"sun":["lunch"]}} \
]'

.PHONY: dev test

dev:
	PORT=$(PORT) \
	TIMEZONE=$(TIMEZONE) \
	PERIODS=$(PERIODS) \
	PEOPLE=$(PEOPLE) \
	ENTRIES=$(ENTRIES) \
	go run ./cmd/anythingsrv

test:
	go test ./... -count=1 -cover -coverprofile /tmp/cover.out
