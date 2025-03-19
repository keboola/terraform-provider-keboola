# Resource Abstraction Layer Usage Guide

This document provides guidance on how to use the new resource abstraction layer for implementing Terraform resources in the Keboola provider.

## Overview

The abstraction layer consists of several components:

1. **ResourceMapper Interface**: Handles mapping between API and Terraform models
2. **NestedResourceHandler Interface**: Manages nested resources within a parent resource
3. **BaseResource**: Provides common CRUD functionality for resources
4. **Utility Functions**: Common operations for JSON handling and nested resources

## How to Implement a New Resource

### 1. Define Your Terraform and API Models

First, define your Terraform model that represents the schema state:

```go
// MyResourceModel represents the Terraform schema for MyResource
type MyResourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    Content     types.String `tfsdk:"configuration"`
    // Add other fields as needed
}
```

### 2. Implement a ResourceMapper

Create a mapper that implements the ResourceMapper interface:

```go
// MyResourceMapper implements ResourceMapper for MyResource
type MyResourceMapper struct {
    // Add any dependencies here
}

// MapAPIToTerraform converts an API model to a Terraform model
func (m *MyResourceMapper) MapAPIToTerraform(
    ctx context.Context,
    apiModel *myapi.MyResource,
    tfModel *MyResourceModel,
) diag.Diagnostics {
    var diags diag.Diagnostics
    
    // Map fields from API to Terraform model
    tfModel.ID = types.StringValue(apiModel.ID)
    tfModel.Name = types.StringValue(apiModel.Name)
    // Map other fields...
    
    return diags
}

// MapTerraformToAPI converts a Terraform model to an API model
func (m *MyResourceMapper) MapTerraformToAPI(
    ctx context.Context,
    tfModel MyResourceModel,
) (*myapi.MyResource, error) {
    // Map fields from Terraform to API model
    apiModel := &myapi.MyResource{
        ID: tfModel.ID.ValueString(),
        Name: tfModel.Name.ValueString(),
        // Map other fields...
    }
    
    return apiModel, nil
}

// ValidateTerraformModel validates a Terraform model
func (m *MyResourceMapper) ValidateTerraformModel(
    ctx context.Context,
    oldModel *MyResourceModel,
    newModel *MyResourceModel,
) diag.Diagnostics {
    var diags diag.Diagnostics
    
    // Validate model fields
    // Check for immutable fields during updates
    if oldModel != nil {
        // Check immutable fields
    }
    
    return diags
}
```

### 3. Implement the Resource

Create your resource implementation using the BaseResource:

```go
// MyResource implements resource.Resource
type MyResource struct {
    base BaseResource[MyResourceModel, *myapi.MyResource]
    client *myapi.Client
}

// Configure sets up the resource
func (r *MyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    
    // Set up the client
    r.client = req.ProviderData.(*providerData).client
    
    // Set up the base resource
    r.base = BaseResource[MyResourceModel, *myapi.MyResource]{
        Mapper: &MyResourceMapper{},
    }
}

// Create creates the resource
func (r *MyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    r.base.ExecuteCreate(ctx, req, resp, func(ctx context.Context, model MyResourceModel) (*myapi.MyResource, error) {
        // Convert Terraform model to API model
        apiModel, err := r.base.Mapper.MapTerraformToAPI(ctx, model)
        if err != nil {
            return nil, err
        }
        
        // Create resource via API
        return r.client.CreateResource(apiModel)
    })
}

// Similar implementations for Read, Update, and Delete
```

## Handling Nested Resources

If your resource contains nested resources, implement a `NestedResourceHandler`:

```go
// MyNestedResourceHandler implements NestedResourceHandler
type MyNestedResourceHandler struct {
    Client *myapi.Client
}

// Implement the required methods for handling nested resources:
// - ExtractChildModels
// - MapChildModelsToAPI
// - ProcessAPIChildModels
```

## Benefits of the Abstraction Layer

1. **Reduced Code Duplication**: Common operations like validation, error handling, and mapping are centralized.
2. **Consistent Error Handling**: Standard approach to handling API errors and diagnostics.
3. **Separation of Concerns**: Clear separation between API interactions, mapping logic, and resource operations.
4. **Type Safety**: Using generics ensures type safety between models.
5. **Testability**: Each component can be tested independently.

## Example Usage in Resource Methods

### Create Operation

```go
func (r *MyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    r.base.ExecuteCreate(ctx, req, resp, func(ctx context.Context, model MyResourceModel) (*myapi.MyResource, error) {
        // Convert and create via API
        apiModel, err := r.base.Mapper.MapTerraformToAPI(ctx, model)
        if err != nil {
            return nil, err
        }
        return r.client.CreateResource(apiModel)
    })
}
```

### Read Operation

```go
func (r *MyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    r.base.ExecuteRead(ctx, req, resp, func(ctx context.Context, model MyResourceModel) (*myapi.MyResource, error) {
        // Read from API using ID or other identifiers
        return r.client.GetResource(model.ID.ValueString())
    })
}
```

### Update Operation

```go
func (r *MyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    r.base.ExecuteUpdate(ctx, req, resp, func(ctx context.Context, state MyResourceModel, plan MyResourceModel) (*myapi.MyResource, error) {
        // Convert plan to API model and update
        apiModel, err := r.base.Mapper.MapTerraformToAPI(ctx, plan)
        if err != nil {
            return nil, err
        }
        return r.client.UpdateResource(apiModel)
    })
}
```

### Delete Operation

```go
func (r *MyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    r.base.ExecuteDelete(ctx, req, resp, func(ctx context.Context, model MyResourceModel) error {
        // Delete via API using ID
        return r.client.DeleteResource(model.ID.ValueString())
    })
}
``` 