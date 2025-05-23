---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "keboola_component_configuration Resource - terraform-provider-keboola"
subcategory: ""
description: |-
  Manages component configurations (https://keboola.docs.apiary.io/#reference/components-and-configurations).
---

# keboola_component_configuration (Resource)

Manages component configurations (https://keboola.docs.apiary.io/#reference/components-and-configurations).

## Example Usage

```terraform
# Manage example configuration.
resource "keboola_component_configuration" "ex_generic_test" {
  name          = "My generic extractor configuration"
  component_id  = "ex-generic-v2"
  description   = "pulls users from my external source"
  is_disabled   = false
  configuration = <<EOT
{
    "parameters": {
        "api": {
            "baseUrl": "http://myexternalresource.com"
        },
        "config": {
            "outputBucket": "output",
            "jobs": [
                {
                    "endpoint": "users",
                    "children": [
                        {
                            "endpoint": "user/{user-id}",
                            "dataField": ".",
                            "placeholders": {
                                "user-id": "id"
                            }
                        }
                    ]
                }
            ]
        }
    }
}
EOT
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `component_id` (String) Id of the component.
- `name` (String) Name of the configuration.

### Optional

- `branch_id` (Number) Id of the branch. If not specified, then default branch will be used.
- `change_description` (String) Change description associated with the configuration change.
- `configuration` (String) Content of the configuration specified as JSON string.
- `configuration_id` (String) Id of the configuration. If not specified, then will be autogenerated.
- `description` (String) Description of the configuration.
- `is_disabled` (Boolean) Wheter configuration is enabled or disabled.
- `rows` (Attributes List) Rows for the configuration (see [below for nested schema](#nestedatt--rows))

### Read-Only

- `created` (String) Timestamp of the configuration creation date.
- `id` (String) Unique string identifier assembled as branchId/componentId/configId.
- `is_deleted` (Boolean) Wheter configuration has been deleted or not.

<a id="nestedatt--rows"></a>
### Nested Schema for `rows`

Required:

- `name` (String) Name of the configuration row.

Optional:

- `change_description` (String) Change description associated with the configuration row change.
- `configuration_row` (String) Content of the configuration row specified as JSON string.
- `description` (String) Description of the configuration row.
- `is_disabled` (Boolean) Whether configuration row is enabled or disabled.

Read-Only:

- `id` (String) ID of the configuration row
- `state` (String) State of the configuration row.


