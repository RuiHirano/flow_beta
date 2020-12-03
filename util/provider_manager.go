package util

import (
	//"log"

	//"sync"

	api "github.com/RuiHirano/flow_beta/api"
)

var (
	//mu sync.Mutex
)

////////////////////////////////////////////////////////////
//////////////       Provider Manager Class      //////////
///////////////////////////////////////////////////////////

type ProviderManager struct {
	MyProvider   *api.Provider
	Providers    []*api.Provider
	ProvidersMap map[api.Provider_Type][]*api.Provider
}

func NewProviderManager(myProvider *api.Provider) *ProviderManager {
	pm := &ProviderManager{
		MyProvider:   myProvider,
		Providers:    []*api.Provider{},
		ProvidersMap: make(map[api.Provider_Type][]*api.Provider),
	}
	return pm
}

func (pm *ProviderManager) AddProvider(p *api.Provider) {
	for _, pv := range pm.Providers {
		if pv.Id == p.Id {
			return
		}
	}
	mu.Lock()
	pm.Providers = append(pm.Providers, p)
	pm.CreateProvidersMap()
	mu.Unlock()
	//log.Printf("Providers: %v\n", pm.Providers)
}

func (pm *ProviderManager) SetProviders(ps []*api.Provider) {
	mu.Lock()
	pm.Providers = ps
	pm.CreateProvidersMap()
	mu.Unlock()
	//log.Printf("Providers: %v\n", pm.Providers)
}

func (pm *ProviderManager) GetProviders() []*api.Provider {
	mu.Lock()
	providers := pm.Providers
	mu.Unlock()
	return providers
	//log.Printf("Providers: %v\n", pm.Providers)
}

func (pm *ProviderManager) DeleteProvider(id uint64) {
	newProviders := make([]*api.Provider, 0)
	for _, provider := range pm.Providers {
		if provider.Id == id {
			continue
		}
		newProviders = append(newProviders, provider)
	}
	pm.Providers = newProviders
	pm.CreateProvidersMap()
}

func (pm *ProviderManager) GetProviderIds(typeList []api.Provider_Type) []uint64 {
	idList := make([]uint64, 0)
	for _, tp := range typeList {
		for _, p := range pm.ProvidersMap[tp] {
			id := p.GetId()
			idList = append(idList, id)
		}
	}
	return idList
}

func (pm *ProviderManager) CreateProvidersMap() {
	providersMap := make(map[api.Provider_Type][]*api.Provider)

	for _, p := range pm.Providers {
		if p.GetId() != pm.MyProvider.GetId() { // 自分は含まない
			switch p.GetType() {
			case api.Provider_MASTER:
				providersMap[api.Provider_MASTER] = append(providersMap[api.Provider_MASTER], p)
			case api.Provider_WORKER:
				providersMap[api.Provider_WORKER] = append(providersMap[api.Provider_WORKER], p)
			case api.Provider_GATEWAY:
				providersMap[api.Provider_GATEWAY] = append(providersMap[api.Provider_GATEWAY], p)
			case api.Provider_VISUALIZATION:
				providersMap[api.Provider_VISUALIZATION] = append(providersMap[api.Provider_VISUALIZATION], p)
			case api.Provider_DATABASE:
				providersMap[api.Provider_DATABASE] = append(providersMap[api.Provider_DATABASE], p)
			case api.Provider_AGENT:
				providersMap[api.Provider_AGENT] = append(providersMap[api.Provider_AGENT], p)
			}
		}
	}
	pm.ProvidersMap = providersMap

}

/*func IsSameArea(area1 *api.Area, area2 *api.Area) bool {
	if area1.GetId() == area2.GetId() {
		// エリアIDが等しければtrue
		return true
	}
	return false
}

// FIX
func IsNeighborArea(area1 *api.Area, area2 *api.Area) bool {
	myControlArea := area1.GetControlArea()
	tControlArea := area2.GetControlArea()
	maxLat, maxLon, minLat, minLon := GetCoordRange(myControlArea)
	tMaxLat, tMaxLon, tMinLat, tMinLon := GetCoordRange(tControlArea)

	if maxLat == tMinLat && (minLon <= tMaxLon && tMaxLon <= maxLon || minLon <= tMinLon && tMinLon <= maxLon) {
		return true
	}
	if minLat == tMaxLat && (minLon <= tMaxLon && tMaxLon <= maxLon || minLon <= tMinLon && tMinLon <= maxLon) {
		return true
	}
	if maxLon == tMinLon && (minLat <= tMaxLat && tMaxLat <= maxLat || minLat <= tMinLat && tMinLat <= maxLat) {
		return true
	}
	if minLon == tMaxLon && (minLat <= tMaxLat && tMaxLat <= maxLat || minLat <= tMinLat && tMinLat <= maxLat) {
		return true
	}
	return false
}

func GetCoordRange(coords []*api.Coord) (float64, float64, float64, float64) {
	maxLon, maxLat := math.Inf(-1), math.Inf(-1)
	minLon, minLat := math.Inf(0), math.Inf(0)
	for _, coord := range coords {
		if coord.Latitude > maxLat {
			maxLat = coord.Latitude
		}
		if coord.Longitude > maxLon {
			maxLon = coord.Longitude
		}
		if coord.Latitude < minLat {
			minLat = coord.Latitude
		}
		if coord.Longitude < minLon {
			minLon = coord.Longitude
		}
	}
	return maxLat, maxLon, minLat, minLon
}
*/
