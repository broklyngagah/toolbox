package toolbox

import (
	"fmt"
	"os"
	"strings"
	"time"
)

//ValueProvider represents a value provider
type ValueProvider interface {
	//Get returns a value for passed in context and arguments. Context can be used to manage state.
	Get(context Context, arguments ...interface{}) (interface{}, error)
}

//ValueProviderRegistry registry of value providers
type ValueProviderRegistry interface {
	Register(name string, valueProvider ValueProvider)

	Contains(name string) bool

	Names() []string

	Get(name string) ValueProvider
}

type valueProviderRegistryImpl struct {
	registry map[string](ValueProvider)
}

func (r valueProviderRegistryImpl) Register(name string, valueProvider ValueProvider) {
	r.registry[name] = valueProvider
}

func (r valueProviderRegistryImpl) Contains(name string) bool {
	_, ok := r.registry[name]
	return ok
}

func (r valueProviderRegistryImpl) Get(name string) ValueProvider {
	if result, ok := r.registry[name]; ok {
		return result
	}
	panic(fmt.Sprintf("failed to lookup name: %v", name))
}

func (r valueProviderRegistryImpl) Names() []string {
	return MapKeysToStringSlice(&r.registry)
}

//NewValueProviderRegistry create new NewValueProviderRegistry
func NewValueProviderRegistry() ValueProviderRegistry {
	var result ValueProviderRegistry = &valueProviderRegistryImpl{
		registry: make(map[string]ValueProvider),
	}
	return result
}

type envValueProvider struct{}

func (p envValueProvider) Get(context Context, arguments ...interface{}) (interface{}, error) {
	key := arguments[0].(string)
	value, found := os.LookupEnv(key)
	if found {
		return value, nil
	}
	return nil, fmt.Errorf("failed to lookup %v in env", key)
}

//NewEnvValueProvider returns a provider that returns a value of env variables.
func NewEnvValueProvider() ValueProvider {
	var result ValueProvider = &envValueProvider{}
	return result
}

type castedValueProvider struct{}

func (p castedValueProvider) Get(context Context, arguments ...interface{}) (interface{}, error) {
	key := arguments[0].(string)
	if len(arguments) < 2 {
		return nil, fmt.Errorf("failed to cast to %v due to invalid number of arguments, Wanted 2 but had:%v", key, len(arguments))
	}
	switch key {
	case "time":
		if len(arguments) != 3 {
			return nil, fmt.Errorf("failed to cast to time due to invalid number of arguments expected 2, but had %v", len(arguments)-1)
		}
		castedTime, err := ParseTime(AsString(arguments[1]), AsString(arguments[2]))
		if err != nil {
			return nil, fmt.Errorf("failed to cast to time %v due to %v", AsString(arguments[1]), err)
		}
		return castedTime, nil
	case "int":
		return AsInt(arguments[1]), nil
	case "float":
		return AsFloat(arguments[1]), nil
	case "bool":
		return AsBoolean(arguments[1]), nil
	case "string":
		return AsString(arguments[1]), nil

	}
	return nil, fmt.Errorf("failed to cast to %v - unsupported type", key)
}

//NewCastedValueProvider return a provider that return casted value type
func NewCastedValueProvider() ValueProvider {
	var result ValueProvider = &castedValueProvider{}
	return result
}

type currentTimeProvider struct{}

func (p currentTimeProvider) Get(context Context, arguments ...interface{}) (interface{}, error) {
	return time.Now(), nil
}

//NewCurrentTimeProvider returns a provder that returns time.Now()
func NewCurrentTimeProvider() ValueProvider {
	var result ValueProvider = &currentTimeProvider{}
	return result
}

type timeDiffProvider struct{}

func (p timeDiffProvider) Get(context Context, arguments ...interface{}) (interface{}, error) {

	var resultTime time.Time
	var durationDelta time.Duration

	if len(arguments) >= 1 {
		if strings.ToLower(AsString(arguments[0])) == "now" {
			resultTime = time.Now()
		} else {
			extractedTime := AsTime(arguments[0], "")
			if extractedTime != nil {
				resultTime = *extractedTime
			}
		}
	}

	if len(arguments) >= 3 {
		var amount = AsInt(arguments[1])
		switch strings.ToLower(AsString(arguments[2])) {
		case "day":
			durationDelta = time.Duration(amount*24) * time.Hour
		case "week":
			durationDelta = time.Duration(amount*24*7) * time.Hour
		case "hour":
			durationDelta = time.Duration(amount) * time.Hour
		case "min":
			durationDelta = time.Duration(amount) * time.Minute
		case "sec":
			durationDelta = time.Duration(amount) * time.Second
		}
	}
	var format = ""
	if len(arguments) == 4 {
		format = AsString(arguments[3])
	}
	resultTime = resultTime.Add(durationDelta)
	switch format {
	case "unix":
		return int(resultTime.Unix()+resultTime.UnixNano()) / 1000000000, nil
	case "timestamp":
		return int(resultTime.Unix()+resultTime.UnixNano()) / 1000000, nil

	default:
		if len(format) > 0 {
			return resultTime.Format(DateFormatToLayout(format)), nil
		}
	}
	return resultTime, nil
}

//NewTimeDiffProvider returns a provider that delta, time unit  and optionally format
//format as java date format or unix or timestamp
func NewTimeDiffProvider() ValueProvider {
	var result ValueProvider = &timeDiffProvider{}
	return result
}

type weekdayProvider struct{}

func (p weekdayProvider) Get(context Context, arguments ...interface{}) (interface{}, error) {
	var now = time.Now()
	return int(now.Weekday()), nil
}

func NewWeekdayProvider() ValueProvider {
	return &weekdayProvider{}
}

type nilValueProvider struct{}

func (p nilValueProvider) Get(context Context, arguments ...interface{}) (interface{}, error) {
	return nil, nil
}

//NewNilValueProvider returns a provider that returns a nil
func NewNilValueProvider() ValueProvider {
	var result ValueProvider = &nilValueProvider{}
	return result
}

type currentDateProvider struct{}

func (p currentDateProvider) Get(context Context, arguments ...interface{}) (interface{}, error) {
	return time.Now().Local().Format("20060102"), nil
}

//NewCurrentDateProvider returns a provider that returns current date in the format yyyymmdd, i.e. 20170205
func NewCurrentDateProvider() ValueProvider {
	var result ValueProvider = &currentDateProvider{}
	return result
}

//Dictionary represents simply lookup interface
type Dictionary interface {
	//Get returns value for passed in key or error
	Get(key string) (interface{}, error)

	//Exists checks if key exists
	Exists(key string) bool
}

//MapDictionary alias to map of string and interface{}
type MapDictionary map[string]interface{}

func (d *MapDictionary) Get(name string) (interface{}, error) {
	if result, found := (*d)[name]; found {
		return result, nil
	}
	return nil, fmt.Errorf("failed to lookup: %v", name)
}

func (d *MapDictionary) Exists(name string) bool {
	_, found := (*d)[name]
	return found
}

type dictionaryProvider struct {
	dictionaryContentKey interface{}
}

func (p dictionaryProvider) Get(context Context, arguments ...interface{}) (interface{}, error) {
	if len(arguments) == 0 {
		return nil, fmt.Errorf("Expected at least one argument but had 0")
	}
	var key = AsString(arguments[0])
	var dictionary Dictionary
	context.GetInto(p.dictionaryContentKey, &dictionary)
	if len(arguments) == 1 && !dictionary.Exists(key) {
		return nil, nil
	}
	return dictionary.Get(key)
}

//NewDictionaryProvider creates a new Dictionary provider, it takes a key context that is a MapDictionary pointer
func NewDictionaryProvider(contextKey interface{}) ValueProvider {
	return &dictionaryProvider{contextKey}
}
