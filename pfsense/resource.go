package pfsense

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

const idSeparator = "."

type updateRequestFunc[RequestType any] func(*schema.ResourceData, string, *RequestType) error
type getFromResourceFunc[ResponseType any] func(*ResponseType) (interface{}, error)

type updateFunc[RequestType any, ResponseType any, IdType ~string | ~int] func(context.Context, *pfsenseapi.Client, IdType, *RequestType) (*ResponseType, error)
type createFunc[RequestType any, ResponseType any] func(context.Context, *pfsenseapi.Client, *RequestType) (*ResponseType, error)
type listFunc[ResponseType any] func(context.Context, *pfsenseapi.Client, string) ([]*ResponseType, error)
type deleteFunc[IdType ~string | ~int] func(context.Context, *pfsenseapi.Client, string, IdType) error
type disableFunc[RequestType any] func(*RequestType) error

var dnsValidator schema.SchemaValidateFunc = validation.StringMatch(regexValidator(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}$`), "Invalid DNS Name")

type resourceProperty[RequestType any, ResponseType any] struct {
	schema          *schema.Schema
	idProperty      bool
	partition       bool
	updateRequest   updateRequestFunc[RequestType]
	getFromResponse getFromResourceFunc[ResponseType]
	validValues     []string
}

type resource[RequestType any, ResponseType any, IdType ~string | ~int] struct {
	name        string
	description string
	getId       func(*ResponseType) (IdType, error)
	partitionId string
	update      updateFunc[RequestType, ResponseType, IdType]
	create      createFunc[RequestType, ResponseType]
	delete      deleteFunc[IdType]
	disable     disableFunc[RequestType]
	list        listFunc[ResponseType]
	properties  map[string]*resourceProperty[RequestType, ResponseType]
}

func (r *resource[RequestType, ResponseType, IdType]) updateRequest(d *schema.ResourceData, request *RequestType) error {
	for name, prop := range r.properties {
		value, exists := d.GetOk(name)

		if exists {
			switch t := value.(type) {
			case string:
				if t == "" {
					exists = false
				}
			case []interface{}:
				if len(t) == 0 {
					exists = false
				}
			}
		}

		if exists {
			if err := prop.updateRequest(d, name, request); err != nil {
				return err
			}
		} else if prop.schema.Default != nil {
			d.Set(name, prop.schema.Default)
		}
	}

	return nil
}

func (r *resource[RequestType, ResponseType, IdType]) updateResource(d *schema.ResourceData, response *ResponseType) error {
	for name, prop := range r.properties {
		value, err := prop.getFromResponse(response)

		if err != nil {
			return err
		}

		if !isNilInterface(value) && value == "" {
			if err = d.Set(name, value); err != nil {
				return err
			}
		} else if prop.schema.Default != nil {
			d.Set(name, prop.schema.Default)
		}
	}

	return nil
}

func (r *resource[RequestType, ResponseType, IdType]) GetCreateFunction() schema.CreateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		client := m.(*pfsenseapi.Client)
		request := new(RequestType)

		if err := r.updateRequest(d, request); err != nil {
			return diag.FromErr(err)
		}

		response, err := r.create(ctx, client, request)

		if err != nil {
			return diag.FromErr(err)
		}

		if err := r.updateResource(d, response); err != nil {
			return diag.FromErr(err)
		}

		id, err := r.getId(response)

		if err != nil {
			return diag.FromErr(err)
		}

		d.SetId(fmt.Sprint(id))

		return nil
	}
}

func (r *resource[RequestType, ResponseType, IdType]) UpdateFromId(ctx context.Context, client *pfsenseapi.Client, d *schema.ResourceData) error {
	var list []*ResponseType
	var err error
	var partition string

	partition, id, err := r.getResourceId(d)

	if err != nil {
		return err
	}

	list, err = r.list(ctx, client, partition)

	if err != nil {
		return err
	}

	for _, item := range list {
		itemId, err := r.getId(item)

		if err != nil {
			return fmt.Errorf("Unable to get Id from listed value, received err: %v", err)
		}

		if id == itemId {
			if err = r.updateResource(d, item); err != nil {
				return err
			}
			return nil
		}
	}

	var partitionErrorText string

	if r.partitionId != "" {
		partitionErrorText = fmt.Sprintf(" and with %s equal to %s", r.partitionId, partition)
	}

	return fmt.Errorf("Unable to find item with Id %s%s", d.Id(), partitionErrorText)

}

func (r *resource[RequestType, ResponseType, IdType]) GetReadFunction() schema.ReadContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		client := m.(*pfsenseapi.Client)
		if err := r.UpdateFromId(ctx, client, d); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}
}

func (r *resource[RequestType, ResponseType, IdType]) getResourceId(d *schema.ResourceData) (string, IdType, error) {
	var partition string
	id := d.Id()

	if r.partitionId != "" {
		parts := strings.Split(id, idSeparator)
		partition = parts[0]
		id = parts[1]
	}

	var zeroValue IdType

	value, ok := any(d.Id()).(IdType)

	if ok {
		return partition, value, nil
	}

	number, err := strconv.Atoi(d.Id())

	if err != nil {
		return "", zeroValue, err
	}

	value, ok = any(number).(IdType)

	if ok {
		return partition, value, nil
	}

	return "", zeroValue, fmt.Errorf("Unable to determine type of Id")
}

func (r *resource[RequestType, ResponseType, IdType]) GetUpdateFunction() schema.UpdateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		client := m.(*pfsenseapi.Client)
		request := new(RequestType)

		if err := r.updateRequest(d, request); err != nil {
			return diag.FromErr(err)
		}

		_, id, err := r.getResourceId(d)

		if err != nil {
			return diag.FromErr(err)
		}

		response, err := r.update(ctx, client, id, request)

		if err != nil {
			return diag.FromErr(err)
		}

		if err = r.updateResource(d, response); err != nil {
			return diag.FromErr(err)
		}

		return nil
	}
}

func (r *resource[RequestType, ResponseType, IdType]) GetDeleteFunction() schema.DeleteContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		client := m.(*pfsenseapi.Client)

		partition, id, err := r.getResourceId(d)

		if err != nil {
			return diag.FromErr(err)
		}

		if r.delete != nil {
			err = r.delete(ctx, client, partition, id)

			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			if err = r.UpdateFromId(ctx, client, d); err != nil {
				return diag.FromErr(err)
			}

			request := new(RequestType)

			if err = r.updateRequest(d, request); err != nil {
				return diag.FromErr(err)
			}

			if err = r.disable(request); err != nil {
				return diag.FromErr(err)
			}

			if _, err = r.update(ctx, client, id, request); err != nil {
				return diag.FromErr(err)
			}
		}

		d.SetId("")

		return nil
	}
}

func (r *resource[RequestType, ResponseType, IdType]) GetImporter() *schema.ResourceImporter {
	return &schema.ResourceImporter{
		StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
			client := m.(*pfsenseapi.Client)

			if err := r.UpdateFromId(ctx, client, d); err != nil {
				return nil, err
			}

			return []*schema.ResourceData{d}, nil
		},
	}
}

func (r *resource[RequestType, ResponseType, IdType]) AddResource(provider *schema.Provider) {
	_, exists := provider.ResourcesMap[r.name]

	if exists {
		panic(fmt.Sprintf("Resource %s already exists", r.name))
	}

	resource := &schema.Resource{
		CreateContext: r.GetCreateFunction(),
		ReadContext:   r.GetReadFunction(),
		UpdateContext: r.GetUpdateFunction(),
		DeleteContext: r.GetDeleteFunction(),
		Importer:      r.GetImporter(),
		Schema:        map[string]*schema.Schema{},
		Description:   r.description,
	}

	var idName string

	for name, property := range r.properties {

		if property.idProperty {
			idName = name
		} else if property.partition {
			r.partitionId = name
		}

		resource.Schema[name] = property.schema
	}

	if idName != "" {
		r.getId = func(response *ResponseType) (IdType, error) {
			var zeroValue IdType
			v, err := r.properties[idName].getFromResponse(response)

			if err != nil {
				return zeroValue, err
			}
			value, ok := v.(IdType)

			if ok {
				return value, nil
			}

			return zeroValue, fmt.Errorf("Unable to convert %v (value of %s) to IdType", v, idName)
		}
	}

	provider.ResourcesMap[r.name] = resource
}
