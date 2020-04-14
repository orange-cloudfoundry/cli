package app

import (
	"crypto/subtle"
	"net/http"
	"time"

	"code.cloudfoundry.org/cli/integration/assets/hydrabroker/config"

	"code.cloudfoundry.org/cli/integration/assets/hydrabroker/store"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pivotal-cf/brokerapi/v7/domain/apiresponses"
)

func brokerCheckRequest(store *store.BrokerConfigurationStore, r *http.Request) (config.BrokerConfiguration, error) {
	guid, err := readGUID(r)
	if err != nil {
		return config.BrokerConfiguration{}, err
	}

	cfg, ok := store.GetBrokerConfiguration(guid)
	if !ok {
		return config.BrokerConfiguration{}, notFoundError{}
	}

	givenUsername, givenPassword, ok := r.BasicAuth()
	if !ok {
		return config.BrokerConfiguration{}, unauthorizedError{}
	}

	// Compare everything every time to protect against timing attacks
	if 2 != subtle.ConstantTimeCompare([]byte(cfg.Username), []byte(givenUsername))+
		subtle.ConstantTimeCompare([]byte(cfg.Password), []byte(givenPassword)) {
		return config.BrokerConfiguration{}, nil
	}

	return cfg, nil
}

func brokerCatalog(store *store.BrokerConfigurationStore, w http.ResponseWriter, r *http.Request) error {
	config, err := brokerCheckRequest(store, r)
	if err != nil {
		return err
	}

	if config.CatalogResponse != 0 {
		w.WriteHeader(config.CatalogResponse)
		return nil
	}

	var services []domain.Service
	for _, s := range config.Services {
		var plans []domain.ServicePlan
		for _, p := range s.Plans {
			var mi *domain.MaintenanceInfo

			if p.MaintenanceInfo != nil {
				mi = &domain.MaintenanceInfo{
					Version:     p.MaintenanceInfo.Version,
					Description: p.MaintenanceInfo.Description,
				}
			}

			plans = append(plans, domain.ServicePlan{
				Name:            p.Name,
				ID:              p.ID,
				Description:     p.Description,
				MaintenanceInfo: mi,
				Bindable:        &s.Bindable,
			})
		}
		services = append(services, domain.Service{
			Name:                s.Name,
			ID:                  s.ID,
			Description:         s.Description,
			Plans:               plans,
			BindingsRetrievable: s.Bindable,
			Metadata: &domain.ServiceMetadata{
				Shareable:        &s.Shareable,
				DocumentationUrl: `http://documentation.url`,
			},
		})
	}

	return respondWithJSON(w, apiresponses.CatalogResponse{Services: services})
}

func brokerProvision(store *store.BrokerConfigurationStore, w http.ResponseWriter, r *http.Request) error {
	config, err := brokerCheckRequest(store, r)
	if err != nil {
		return err
	}

	if config.ProvisionResponse != 0 {
		w.WriteHeader(config.ProvisionResponse)
		return nil
	}

	response := map[string]interface{}{
		"dashboard_url": `http://example.com`,
	}

	switch config.AsyncResponseDelay {
	case 0:
		w.WriteHeader(http.StatusCreated)
		return respondWithJSON(w, response)
	default:
		return brokerAsyncResponse(w, config.AsyncResponseDelay, response)
	}
}

func brokerDeprovision(store *store.BrokerConfigurationStore, w http.ResponseWriter, r *http.Request) error {
	config, err := brokerCheckRequest(store, r)
	if err != nil {
		return err
	}

	if config.DeprovisionResponse != 0 {
		w.WriteHeader(config.DeprovisionResponse)
		return nil
	}

	switch config.AsyncResponseDelay {
	case 0:
		w.WriteHeader(http.StatusOK)
		return respondWithJSON(w, map[string]interface{}{})
	default:
		return brokerAsyncResponse(w, config.AsyncResponseDelay, nil)
	}
}

func brokerBind(store *store.BrokerConfigurationStore, w http.ResponseWriter, r *http.Request) error {
	config, err := brokerCheckRequest(store, r)
	if err != nil {
		return err
	}

	if config.BindResponse != 0 {
		w.WriteHeader(config.BindResponse)
		return nil
	}

	switch config.AsyncResponseDelay {
	case 0:
		w.WriteHeader(http.StatusCreated)
		return respondWithJSON(w, map[string]interface{}{})
	default:
		return brokerAsyncResponse(w, config.AsyncResponseDelay, nil)
	}
}

func brokerGetBinding(store *store.BrokerConfigurationStore, w http.ResponseWriter, r *http.Request) error {
	config, err := brokerCheckRequest(store, r)
	if err != nil {
		return err
	}

	if config.GetBindingResponse != 0 {
		w.WriteHeader(config.GetBindingResponse)
		return nil
	}

	return respondWithJSON(w, map[string]interface{}{})
}

func brokerUnbind(store *store.BrokerConfigurationStore, w http.ResponseWriter, r *http.Request) error {
	config, err := brokerCheckRequest(store, r)
	if err != nil {
		return err
	}

	if config.UnbindResponse != 0 {
		w.WriteHeader(config.UnbindResponse)
		return nil
	}

	switch config.AsyncResponseDelay {
	case 0:
		w.WriteHeader(http.StatusOK)
		return respondWithJSON(w, map[string]interface{}{})
	default:
		return brokerAsyncResponse(w, config.AsyncResponseDelay, nil)
	}
}

func brokerAsyncResponse(w http.ResponseWriter, duration time.Duration, params map[string]interface{}) error {
	if params == nil {
		params = make(map[string]interface{})
	}

	params["operation"] = time.Now().Add(duration)

	w.WriteHeader(http.StatusAccepted)
	return respondWithJSON(w, params)
}

func brokerLastOperation(store *store.BrokerConfigurationStore, w http.ResponseWriter, r *http.Request) error {
	_, err := brokerCheckRequest(store, r)
	if err != nil {
		return err
	}

	var when time.Time
	if err := when.UnmarshalJSON([]byte(`"` + r.FormValue("operation") + `"`)); err != nil {
		return err
	}

	var result apiresponses.LastOperationResponse
	switch time.Now().After(when) {
	case true:
		result.State = domain.Succeeded
	case false:
		result.State = domain.InProgress
	}

	return respondWithJSON(w, result)
}
