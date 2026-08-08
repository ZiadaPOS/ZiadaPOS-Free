// Package accounts implements users, organisations, stores and auth.
// tanzania.go holds static Tanzania reference data used across registration,
// store setup and filtering.
package accounts

// BusinessType describes a kind of shop/business. The platform supports every
// shop type found in Tanzania - retail, wholesale, pharmacy and more.
type BusinessType struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	NameSwahili string `json:"name_swahili"`
	Description string `json:"description"`
}

// BusinessTypes is the full catalogue of supported shop types.
var BusinessTypes = []BusinessType{
	{Slug: "retail", Name: "Retail Shop (Duka la Rejareja)", NameSwahili: "Duka la Rejareja", Description: "Sells goods directly to final consumers"},
	{Slug: "wholesale", Name: "Wholesale (Jumla)", NameSwahili: "Duka la Jumla", Description: "Sells goods in bulk to retailers"},
	{Slug: "pharmacy", Name: "Pharmacy (Duka la Dawa Muhimu)", NameSwahili: "Duka la Dawa Muhimu", Description: "Sells medicines and health products"},
	{Slug: "general_store", Name: "General Store (Duka la Kawaida)", NameSwahili: "Duka la Kawaida", Description: "Mixed everyday goods"},
	{Slug: "grocery", Name: "Grocery Shop", NameSwahili: "Duka la Mboga", Description: "Foodstuffs, vegetables and provisions"},
	{Slug: "supermarket", Name: "Supermarket", NameSwahili: "Supamaketi", Description: "Large self-service store"},
	{Slug: "hardware", Name: "Hardware (Vifaa vya Ujenzi)", NameSwahili: "Duka la Vifaa", Description: "Building materials and tools"},
	{Slug: "electronics", Name: "Electronics Shop", NameSwahili: "Duka la Umeme", Description: "Electrical and electronic goods"},
	{Slug: "fashion", Name: "Boutique / Fashion", NameSwahili: "Boutique", Description: "Clothing and accessories"},
	{Slug: "shoes", Name: "Shoes Shop", NameSwahili: "Duka la Viatu", Description: "Footwear and shoes"},
	{Slug: "restaurant", Name: "Restaurant / Food", NameSwahili: "Mkahawa", Description: "Prepared food and drinks"},
	{Slug: "cafe", Name: "Cafe / Coffee Shop", NameSwahili: "Duka la Kahawa", Description: "Coffee and light meals"},
	{Slug: "salon", Name: "Salon / Beauty", NameSwahili: "Saluni", Description: "Hair and beauty services"},
	{Slug: "barbershop", Name: "Barbershop", NameSwahili: "Kinyozi", Description: "Barber services"},
	{Slug: "butchery", Name: "Butchery (Mchinjaji)", NameSwahili: "Duka la Nyama", Description: "Meat sales"},
	{Slug: "bakery", Name: "Bakery", NameSwahili: "Bakery", Description: "Bread and baked goods"},
	{Slug: "mobile_shop", Name: "Mobile / Phone Shop", NameSwahili: "Duka la Simu", Description: "Phones and accessories"},
	{Slug: "stationery", Name: "Stationery / Bookshop", NameSwahili: "Duka la Vitabu", Description: "Books and office supplies"},
	{Slug: "fuel_station", Name: "Fuel / Petrol Station", NameSwahili: "Kituo cha Mafuta", Description: "Fuel and service station"},
	{Slug: "cosmetics", Name: "Cosmetics & Toiletries", NameSwahili: "Duka la Vipodozi", Description: "Beauty and personal care products"},
	{Slug: "agriculture", Name: "Agri Inputs / Seeds", NameSwahili: "Duka la Kilimo", Description: "Farm inputs and produce"},
	{Slug: "other", Name: "Other Business", NameSwahili: "Biashara Nyingine", Description: "Any other type of shop or business"},
}

// Occupation describes the user's job. Registration asks the user what they
// do so we can tailor onboarding, pricing and AI assistance.
type Occupation struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Occupations is the list offered during registration.
var Occupations = []Occupation{
	{Slug: "shop_owner", Name: "Shop Owner / Proprietor"},
	{Slug: "retailer", Name: "Retailer"},
	{Slug: "wholesaler", Name: "Wholesaler"},
	{Slug: "pharmacist", Name: "Pharmacist"},
	{Slug: "trader", Name: "Trader"},
	{Slug: "entrepreneur", Name: "Entrepreneur"},
	{Slug: "business_manager", Name: "Business Manager"},
	{Slug: "accountant", Name: "Accountant / Bookkeeper"},
	{Slug: "salesperson", Name: "Salesperson / Cashier"},
	{Slug: "student", Name: "Student"},
	{Slug: "other", Name: "Other"},
}

// ValidBusinessType reports whether slug is a known business type.
func ValidBusinessType(slug string) bool {
	for _, b := range BusinessTypes {
		if b.Slug == slug {
			return true
		}
	}
	return false
}

// ValidOccupation reports whether slug is a known occupation.
func ValidOccupation(slug string) bool {
	for _, o := range Occupations {
		if o.Slug == slug {
			return true
		}
	}
	return false
}

// Region holds a Tanzania administrative region with its districts.
type Region struct {
	Slug     string   `json:"slug"`
	Name     string   `json:"name"`
	Districts []string `json:"districts"`
}

// Ward is a sub-district locality. Only major wards are listed; users may
// also type a custom ward.
type Ward struct {
	Region  string
	District string
	Name    string
}

// Regions covers all 31 Tanzania regions with their districts.
var Regions = []Region{
	{Slug: "arusha", Name: "Arusha", Districts: []string{"Arusha City", "Arusha DC", "Karatu", "Longido", "Meru", "Monduli", "Ngorongoro"}},
	{Slug: "dar-es-salaam", Name: "Dar es Salaam", Districts: []string{"Ilala", "Kigamboni", "Kinondoni", "Temeke", "Ubungo"}},
	{Slug: "dodoma", Name: "Dodoma", Districts: []string{"Dodoma City", "Bahi", "Chamwino", "Chemba", "Kondoa", "Kongwa", "Mpwapwa"}},
	{Slug: "geita", Name: "Geita", Districts: []string{"Geita", "Bukombe", "Chato", "Mbogwe", "Nyang'hwale"}},
	{Slug: "iringa", Name: "Iringa", Districts: []string{"Iringa City", "Iringa DC", "Kilolo", "Mufindi"}},
	{Slug: "kagera", Name: "Kagera", Districts: []string{"Bukoba", "Biharamulo", "Karagwe", "Kyerwa", "Missenyi", "Muleba", "Ngara"}},
	{Slug: "katavi", Name: "Katavi", Districts: []string{"Mpanda", "Mlele", "Moba", "Nkasi"}},
	{Slug: "kigoma", Name: "Kigoma", Districts: []string{"Kigoma City", "Kigoma DC", "Buhigwe", "Kakonko", "Kasulu", "Kibondo", "Uvinza"}},
	{Slug: "kilimanjaro", Name: "Kilimanjaro", Districts: []string{"Moshi City", "Moshi DC", "Hai", "Mwanga", "Rombo", "Same", "Siha"}},
	{Slug: "lindi", Name: "Lindi", Districts: []string{"Lindi", "Kilwa", "Liwale", "Nachingwea", "Ruangwa"}},
	{Slug: "manyara", Name: "Manyara", Districts: []string{"Babati", "Hanang", "Kiteto", "Mbulu", "Simanjiro"}},
	{Slug: "mara", Name: "Mara", Districts: []string{"Musoma", "Bunda", "Butiama", "Rorya", "Serengeti", "Tarime"}},
	{Slug: "mbeya", Name: "Mbeya", Districts: []string{"Mbeya City", "Busokelo", "Chunya", "Kyela", "Mbarali", "Mbozi", "Rungwe"}},
	{Slug: "morogoro", Name: "Morogoro", Districts: []string{"Morogoro City", "Gairo", "Kilombero", "Kilosa", "Malinyi", "Mvomero", "Ulanga"}},
	{Slug: "mtwara", Name: "Mtwara", Districts: []string{"Mtwara City", "Mtwara DC", "Masasi", "Nanyumbu", "Newala", "Tandahimba"}},
	{Slug: "mwanza", Name: "Mwanza", Districts: []string{"Mwanza City", "Buchosa", "Ilemela", "Kwimba", "Magu", "Misungwi", "Nyamagana", "Sengerema", "Ukerewe"}},
	{Slug: "njombe", Name: "Njombe", Districts: []string{"Njombe", "Ludewa", "Makambako", "Makete", "Wanging'ombe"}},
	{Slug: "pwani", Name: "Pwani", Districts: []string{"Kibaha", "Bagamoyo", "Chalinze", "Kibiti", "Kisarawe", "Mafia", "Mkuranga", "Rufiji"}},
	{Slug: "rukwa", Name: "Rukwa", Districts: []string{"Sumbawanga", "Kalambo", "Nkasi"}},
	{Slug: "ruvuma", Name: "Ruvuma", Districts: []string{"Songea", "Mbinga", "Namtumbo", "Nyasa", "Tunduru"}},
	{Slug: "shinyanga", Name: "Shinyanga", Districts: []string{"Shinyanga City", "Shinyanga DC", "Kahama", "Kishapu"}},
	{Slug: "simiyu", Name: "Simiyu", Districts: []string{"Bariadi", "Busega", "Itilima", "Maswa", "Meatu"}},
	{Slug: "singida", Name: "Singida", Districts: []string{"Singida", "Ikungi", "Iramba", "Itigi", "Manyoni", "Mkalama"}},
	{Slug: "songwe", Name: "Songwe", Districts: []string{"Vwawa", "Ileje", "Momba", "Songwe", "Tunduma"}},
	{Slug: "tabora", Name: "Tabora", Districts: []string{"Tabora City", "Igunga", "Kaliua", "Nzega", "Sikonge", "Urambo", "Uyui"}},
	{Slug: "tanga", Name: "Tanga", Districts: []string{"Tanga City", "Handeni", "Kilindi", "Korogwe", "Lushoto", "Mkinga", "Muheza", "Pangani"}},
	{Slug: "unguja-north", Name: "Unguja North", Districts: []string{"Kaskazini A", "Kaskazini B"}},
	{Slug: "unguja-south", Name: "Unguja South", Districts: []string{"Kati", "Kusini"}},
	{Slug: "unguja-west", Name: "Unguja Urban West", Districts: []string{"Magharibi", "Mjini"}},
	{Slug: "pemba-north", Name: "Pemba North", Districts: []string{"Micheweni", "Wete"}},
	{Slug: "pemba-south", Name: "Pemba South", Districts: []string{"Chake Chake", "Mkoani"}},
}

// RegionNames is a quick slug -> name lookup.
var RegionNames = func() map[string]string {
	m := make(map[string]string)
	for _, r := range Regions {
		m[r.Slug] = r.Name
	}
	return m
}()

// ValidRegion reports whether slug is a known region.
func ValidRegion(slug string) bool {
	_, ok := RegionNames[slug]
	return ok
}

// RegionDistricts returns the districts of a region (nil if unknown).
func RegionDistricts(slug string) []string {
	for _, r := range Regions {
		if r.Slug == slug {
			return r.Districts
		}
	}
	return nil
}
