# Telemetry Configuration Example

This example demonstrates how to set up a Generic Extractor configuration with multiple configuration rows for collecting telemetry data using Terraform and the Keboola Provider.

## Configuration Structure

The example includes:
1. A main Generic Extractor configuration (`telemetry_extractor`)
2. Two configuration rows:
   - Daily metrics collection
   - Hourly metrics collection

## Usage

1. Set up your credentials:
   ```bash
   export KBC_HOST="your-keboola-stack-url"
   export KBC_TOKEN="your-storage-api-token"
   ```

2. Initialize Terraform:
   ```bash
   terraform init
   ```

3. Review the planned changes:
   ```bash
   terraform plan
   ```

4. Apply the configuration:
   ```bash
   terraform apply
   ```

## Configuration Details

- The main configuration sets up a Generic Extractor with basic API settings
- Configuration rows define specific data collection schedules:
  - Daily metrics collection with daily period settings
  - Hourly metrics collection with hourly period settings
- Each configuration includes data filtering and processing settings

## Customization

To customize this example for your needs:
1. Modify the `baseUrl` in the main configuration
2. Adjust the API authentication settings
3. Update the configuration rows' parameters according to your API requirements 