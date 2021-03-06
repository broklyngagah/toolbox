package storage

import (
	"fmt"
	"io"
	"net/url"
)

//Service represents abstract way to accessing local or remote storage
type Service interface {
	//List returns a list of object for supplied url
	List(URL string) ([]Object, error)

	//Exists returns true if resource exists
	Exists(URL string) (bool, error)

	//Object returns a Object for supplied url
	StorageObject(URL string) (Object, error)

	//Download returns reader for downloaded storage object
	Download(object Object) (io.Reader, error)

	//Upload uploads provided reader content for supplied storage object.
	Upload(URL string, reader io.Reader) error

	//Delete removes passed in storage object
	Delete(object Object) error

	//Register register schema with provided service
	Register(schema string, service Service) error

	//Closes storage service
	Close() error
}

type storageService struct {
	registry map[string]Service
}

func (s *storageService) getServiceForSchema(URL string) (Service, error) {
	parsedUrl, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	if result, found := s.registry[parsedUrl.Scheme]; found {
		return result, nil
	}

	return nil, fmt.Errorf("failed to lookup url schema %v in %v", parsedUrl.Scheme, URL)
}

//List lists all object for passed in URL
func (s *storageService) List(URL string) ([]Object, error) {
	service, err := s.getServiceForSchema(URL)
	if err != nil {
		return nil, err
	}
	return service.List(URL)
}

//Exists returns true if resource exists
func (s *storageService) Exists(URL string) (bool, error) {
	service, err := s.getServiceForSchema(URL)
	if err != nil {
		return false, err
	}
	return service.Exists(URL)
}

//StorageObject returns storage object for provided URL
func (s *storageService) StorageObject(URL string) (Object, error) {
	service, err := s.getServiceForSchema(URL)
	if err != nil {
		return nil, err
	}
	return service.StorageObject(URL)
}

//Download downloads content for passed in object
func (s *storageService) Download(object Object) (io.Reader, error) {
	service, err := s.getServiceForSchema(object.URL())
	if err != nil {
		return nil, err
	}
	return service.Download(object)
}

//Uploads content for passed in URL
func (s *storageService) Upload(URL string, reader io.Reader) error {
	service, err := s.getServiceForSchema(URL)
	if err != nil {
		return err
	}
	return service.Upload(URL, reader)
}

//Delete remove storage object
func (s *storageService) Delete(object Object) error {
	service, err := s.getServiceForSchema(object.URL())
	if err != nil {
		return err
	}
	return service.Delete(object)
}

//Close closes resources
func (s *storageService) Close() error {
	for _, service := range s.registry {
		err := service.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

//Register register storage schema
func (s *storageService) Register(schema string, service Service) error {
	s.registry[schema] = service
	return nil
}

//NewService creates a new storage service
func NewService() Service {
	var result = &storageService{
		registry: make(map[string]Service),
	}
	result.Register("file", &fileStorageService{})
	result.Register("mem", NewMemoryService())
	return result
}

//NewServiceForURL creates a new storage service for provided URL scheme and optional credential file
func NewServiceForURL(URL, credentialFile string) (Service, error) {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	service := NewService()
	provider := NewStorageProvider().Get(parsedURL.Scheme)

	if provider != nil {
		serviceForScheme, err := provider(credentialFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get storage for url %v: %v", URL, err)
		}
		err = service.Register(parsedURL.Scheme, serviceForScheme)
		if err != nil {
			return nil, err
		}
	} else if parsedURL.Scheme != "file" {
		return nil, fmt.Errorf("unsupported scheme %v", URL)
	}
	return service, nil
}
