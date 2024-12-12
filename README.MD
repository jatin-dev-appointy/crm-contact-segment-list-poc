# Segment List Rule POC

### Pre - Reqiurements

- Rules to be added in `demo.segment_rule` (waqt-dev database)

- Below is the sample rule
```
{
  "combinator": "OR",
  "rules": [
    {
      "field": "phone_number",
      "operator": "LIKE",
      "value": "91%"
    },
    {
      "field": "email",
      "operator": "LIKE",
      "value": "%jay%"
    }
  ]
}
```
- E.g. Query to add rules
```
INSERT INTO demo.segment_rule (name, rule)
VALUES ('Rule3', '{
    "combinator": "AND",
    "rules": [
        {
              "field": "first_name",
              "operator": "IN",
              "value": ["Customer", "Jay"]
            },
        {
              "field": "email",
              "operator": "LIKE",
              "value": "%gmail%"
        }
    ]
}'::jsonb);
```

- When all the rules are added, Our Program will fetch all the rules from the database.
- It will list customers based each rule. (for the given company id).

### Steps to Run

- Run the file
- Enter CompanyId (in which you need to apply rules)
- Enter Database string to connect