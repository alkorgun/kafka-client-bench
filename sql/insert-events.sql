INSERT INTO streams.events_producer
SELECT *,
    toString(id) as `_key`
from faker.events
limit 10000000
SETTINGS
	output_format_json_quote_64bit_integers=0,
    date_time_output_format='iso';
