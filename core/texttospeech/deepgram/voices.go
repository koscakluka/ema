package deepgram

const defaultVoice = VoiceAuraAngus

func GetAvailableVoices() []deepgramVoice {
	return []deepgramVoice{
		VoiceAuraAsteria,
		VoiceAuraLuna,
		VoiceAuraStella,
		VoiceAuraAthena,
		VoiceAuraHera,
		VoiceAuraOrion,
		VoiceAuraArcas,
		VoiceAuraPerseus,
		VoiceAuraAngus,
		VoiceAuraOrpheus,
		VoiceAuraHelios,
		VoiceAuraZeus,
		VoiceAura2Amalthea,
		VoiceAura2Andromeda,
		VoiceAura2Apollo,
		VoiceAura2Arcas,
		VoiceAura2Aries,
		VoiceAura2Asteria,
		VoiceAura2Athena,
		VoiceAura2Atlas,
		VoiceAura2Aurora,
		VoiceAura2Callista,
		VoiceAura2Cora,
		VoiceAura2Cordelia,
		VoiceAura2Delia,
		VoiceAura2Draco,
		VoiceAura2Electra,
		VoiceAura2Harmonia,
		VoiceAura2Helena,
		VoiceAura2Hera,
		VoiceAura2Hermes,
		VoiceAura2Hyperion,
		VoiceAura2Iris,
		VoiceAura2Janus,
		VoiceAura2Juno,
		VoiceAura2Jupiter,
		VoiceAura2Luna,
		VoiceAura2Mars,
		VoiceAura2Minerva,
		VoiceAura2Neptune,
		VoiceAura2Odysseus,
		VoiceAura2Ophelia,
		VoiceAura2Orion,
		VoiceAura2Orpheus,
		VoiceAura2Pandora,
		VoiceAura2Phoebe,
		VoiceAura2Pluto,
		VoiceAura2Saturn,
		VoiceAura2Selene,
		VoiceAura2Thalia,
		VoiceAura2Theia,
		VoiceAura2Vesta,
		VoiceAura2Zeus,
	}
}

func GetVoiceInfo(voice deepgramVoice) voiceInfo {
	resp, ok := Voices[voice]
	if !ok {
		return voiceInfo{}
	}

	return resp
}

type deepgramVoice string

const (
	// Female English (US)
	VoiceAuraAsteria deepgramVoice = "aura-asteria-en"

	// Female English (US)
	VoiceAuraLuna deepgramVoice = "aura-luna-en"

	// Female English (US)
	VoiceAuraStella deepgramVoice = "aura-stella-en"

	// Female English (UK)
	VoiceAuraAthena deepgramVoice = "aura-athena-en"

	// Female English (US)
	VoiceAuraHera deepgramVoice = "aura-hera-en"

	// Male English (US)
	VoiceAuraOrion deepgramVoice = "aura-orion-en"

	// Male English (US)
	VoiceAuraArcas deepgramVoice = "aura-arcas-en"

	// Male English (US)
	VoiceAuraPerseus deepgramVoice = "aura-perseus-en"

	// Male English (Ireland)
	VoiceAuraAngus deepgramVoice = "aura-angus-en"

	// Male English (US)
	VoiceAuraOrpheus deepgramVoice = "aura-orpheus-en"

	// Male English (UK)
	VoiceAuraHelios deepgramVoice = "aura-helios-en"

	// Male English (UK)
	VoiceAuraZeus deepgramVoice = "aura-zeus-en"

	// English (PH)
	//
	// Young Adult Female
	//
	// Engaging, Natural, Cheerful
	//
	// Casual chat
	VoiceAura2Amalthea deepgramVoice = "aura-2-amalthea-en"

	// English (US)
	//
	// Adult Female
	//
	// Casual, Expressive, Comfortable
	//
	// Customer service, IVR
	VoiceAura2Andromeda deepgramVoice = "aura-2-andromeda-en"

	// English (US)
	//
	// Adult Male
	//
	// Confident, Comfortable, Casual
	//
	// Casual chat
	VoiceAura2Apollo deepgramVoice = "aura-2-apollo-en"

	// English (US)
	//
	// Adult Male
	//
	// Natural, Smooth, Clear, Comfortable
	//
	// Customer service, casual chat
	VoiceAura2Arcas deepgramVoice = "aura-2-arcas-en"

	// English (US)
	//
	// Adult Male
	//
	// Warm, Energetic, Caring
	//
	// Casual chat
	VoiceAura2Aries deepgramVoice = "aura-2-aries-en"

	// English (US)
	//
	// Adult Female
	//
	// Clear, Confident, Knowledgeable, Energetic
	//
	// Advertising
	VoiceAura2Asteria deepgramVoice = "aura-2-asteria-en"

	// English (US)
	//
	// Mature Female
	//
	// Calm, Smooth, Professional
	//
	// Storytelling
	VoiceAura2Athena deepgramVoice = "aura-2-athena-en"

	// English (US)
	//
	// Mature Male
	//
	// Enthusiastic, Confident, Approachable, Friendly
	//
	// Advertising
	VoiceAura2Atlas deepgramVoice = "aura-2-atlas-en"

	// English (US)
	//
	// Adult Female
	//
	// Cheerful, Expressive, Energetic
	//
	// Interview
	VoiceAura2Aurora deepgramVoice = "aura-2-aurora-en"

	// English (US)
	//
	// Adult Female
	//
	// Clear, Energetic, Professional, Smooth
	//
	// IVR
	VoiceAura2Callista deepgramVoice = "aura-2-callista-en"

	// English (US)
	//
	// Adult Female
	//
	// Smooth, Melodic, Caring
	//
	// Storytelling
	VoiceAura2Cora deepgramVoice = "aura-2-cora-en"

	// English (US)
	//
	// Young Adult
	//
	// Approachable, Warm, Polite
	//
	// Storytelling
	VoiceAura2Cordelia deepgramVoice = "aura-2-cordelia-en"

	// English (US)
	//
	// Young Adult
	//
	// Casual, Friendly, Cheerful, Breathy
	//
	// Interview
	VoiceAura2Delia deepgramVoice = "aura-2-delia-en"

	// English (GB)
	//
	// Adult Male
	//
	// Warm, Approachable, Trustworthy, Baritone
	//
	// Storytelling
	VoiceAura2Draco deepgramVoice = "aura-2-draco-en"

	// English (US)
	//
	// Adult Female
	//
	// Professional, Engaging, Knowledgeable
	//
	// IVR, advertising, customer service
	VoiceAura2Electra deepgramVoice = "aura-2-electra-en"

	// English (US)
	//
	// Adult Female
	//
	// Empathetic, Clear, Calm, Confident
	//
	// Customer service
	VoiceAura2Harmonia deepgramVoice = "aura-2-harmonia-en"

	// English (US)
	//
	// Adult Female
	//
	// Caring, Natural, Positive, Friendly, Raspy
	//
	// IVR, casual chat
	VoiceAura2Helena deepgramVoice = "aura-2-helena-en"

	// English (US)
	//
	// Adult Female
	//
	// Smooth, Warm, Professional
	//
	// Informative
	VoiceAura2Hera deepgramVoice = "aura-2-hera-en"

	// English (US)
	//
	// Adult Male
	//
	// Expressive, Engaging, Professional
	//
	// Informative
	VoiceAura2Hermes deepgramVoice = "aura-2-hermes-en"

	// English (AU)
	//
	// Adult Male
	//
	// Caring, Warm, Empathetic
	//
	// Interview
	VoiceAura2Hyperion deepgramVoice = "aura-2-hyperion-en"

	// English (US)
	//
	// Young Adult
	//
	// Cheerful, Positive, Approachable
	//
	// IVR, advertising, customer service
	VoiceAura2Iris deepgramVoice = "aura-2-iris-en"

	// English (US)
	//
	// Adult Female
	//
	// Southern, Smooth, Trustworthy
	//
	// Storytelling
	VoiceAura2Janus deepgramVoice = "aura-2-janus-en"

	// English (US)
	//
	// Adult Female
	//
	// Natural, Engaging, Melodic, Breathy
	//
	// Interview
	VoiceAura2Juno deepgramVoice = "aura-2-juno-en"

	// English (US)
	//
	// Adult Male
	//
	// Expressive, Knowledgeable, Baritone
	//
	// Informative
	VoiceAura2Jupiter deepgramVoice = "aura-2-jupiter-en"

	// English (US)
	//
	// Young Adult
	//
	// Friendly, Natural, Engaging
	//
	// IVR
	VoiceAura2Luna deepgramVoice = "aura-2-luna-en"

	// English (US)
	//
	// Adult Male
	//
	// Smooth, Patient, Trustworthy, Baritone
	//
	// Customer service
	VoiceAura2Mars deepgramVoice = "aura-2-mars-en"

	// English (US)
	//
	// Adult Female
	//
	// Positive, Friendly, Natural
	//
	// Storytelling
	VoiceAura2Minerva deepgramVoice = "aura-2-minerva-en"

	// English (US)
	//
	// Adult Male
	//
	// Professional, Patient, Polite
	//
	// Customer service
	VoiceAura2Neptune deepgramVoice = "aura-2-neptune-en"

	// English (US)
	//
	// Adult Male
	//
	// Calm, Smooth, Comfortable, Professional
	//
	// Advertising
	VoiceAura2Odysseus deepgramVoice = "aura-2-odysseus-en"

	// English (US)
	//
	// Adult Female
	//
	// Expressive, Enthusiastic, Cheerful
	//
	// Interview
	VoiceAura2Ophelia deepgramVoice = "aura-2-ophelia-en"

	// English (US)
	//
	// Adult Male
	//
	// Approachable, Comfortable, Calm, Polite
	//
	// Informative
	VoiceAura2Orion deepgramVoice = "aura-2-orion-en"

	// English (US)
	//
	// Adult Male
	//
	// Professional, Clear, Confident, Trustworthy
	//
	// Customer service, storytelling
	VoiceAura2Orpheus deepgramVoice = "aura-2-orpheus-en"

	// English (GB)
	//
	// Adult Female
	//
	// Smooth, Calm, Melodic, Breathy
	//
	// IVR, informative
	VoiceAura2Pandora deepgramVoice = "aura-2-pandora-en"

	// English (US)
	//
	// Adult Female
	//
	// Energetic, Warm, Casual
	//
	// Customer service
	VoiceAura2Phoebe deepgramVoice = "aura-2-phoebe-en"

	// English (US)
	//
	// Adult Male
	//
	// Smooth, Calm, Empathetic, Baritone
	//
	// Interview, storytelling
	VoiceAura2Pluto deepgramVoice = "aura-2-pluto-en"

	// English (US)
	//
	// Adult Male
	//
	// Knowledgeable, Confident, Baritone
	//
	// Customer service
	VoiceAura2Saturn deepgramVoice = "aura-2-saturn-en"

	// English (US)
	//
	// Adult Female
	//
	// Expressive, Engaging, Energetic
	//
	// Informative
	VoiceAura2Selene deepgramVoice = "aura-2-selene-en"

	// English (US)
	//
	// Adult Female
	//
	// Clear, Confident, Energetic, Enthusiastic
	//
	// Casual chat, customer service, IVR
	VoiceAura2Thalia deepgramVoice = "aura-2-thalia-en"

	// English (AU)
	//
	// Adult Female
	//
	// Expressive, Polite, Sincere
	//
	// Informative
	VoiceAura2Theia deepgramVoice = "aura-2-theia-en"

	// English (US)
	//
	// Adult Female
	//
	// Natural, Expressive, Patient, Empathetic
	//
	// Customer service, interview, storytelling
	VoiceAura2Vesta deepgramVoice = "aura-2-vesta-en"

	// English (US)
	//
	// Adult Male
	//
	// Deep, Trustworthy, Smooth
	//
	// IVR
	VoiceAura2Zeus deepgramVoice = "aura-2-zeus-en"
)

type voiceInfo struct {
	ID              deepgramVoice
	Name            string
	Model           string
	Age             string
	Gender          string
	Language        string
	Locale          string
	Characteristics string
	UseCases        string
}

var Voices = map[deepgramVoice]voiceInfo{
	VoiceAuraAsteria: {
		ID:              VoiceAuraAsteria,
		Name:            "Asteria",
		Model:           "aura-asteria-en",
		Age:             "",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraLuna: {
		ID:              VoiceAuraLuna,
		Name:            "Luna",
		Model:           "aura-luna-en",
		Age:             "",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraStella: {
		ID:              VoiceAuraStella,
		Name:            "Stella",
		Model:           "aura-stella-en",
		Age:             "",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraAthena: {
		ID:              VoiceAuraAthena,
		Name:            "Athena",
		Model:           "aura-athena-en",
		Age:             "",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_gb",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraHera: {
		ID:              VoiceAuraHera,
		Name:            "Hera",
		Model:           "aura-hera-en",
		Age:             "",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraOrion: {
		ID:              VoiceAuraOrion,
		Name:            "Orion",
		Model:           "aura-orion-en",
		Age:             "",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraArcas: {
		ID:              VoiceAuraArcas,
		Name:            "Arcas",
		Model:           "aura-arcas-en",
		Age:             "",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraPerseus: {
		ID:              VoiceAuraPerseus,
		Name:            "Perseus",
		Model:           "aura-perseus-en",
		Age:             "",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraAngus: {
		ID:              VoiceAuraAngus,
		Name:            "Angus",
		Model:           "aura-angus-en",
		Age:             "",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_ie",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraOrpheus: {
		ID:              VoiceAuraOrpheus,
		Name:            "Orpheus",
		Model:           "aura-orpheus-en",
		Age:             "",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraHelios: {
		ID:              VoiceAuraHelios,
		Name:            "Helios",
		Model:           "aura-helios-en",
		Age:             "",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_gb",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAuraZeus: {
		ID:              VoiceAuraZeus,
		Name:            "Zeus",
		Model:           "aura-zeus-en",
		Age:             "",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_gb",
		Characteristics: "",
		UseCases:        "",
	},
	VoiceAura2Amalthea: {
		ID:              VoiceAura2Amalthea,
		Name:            "Amalthea",
		Model:           "aura-2-amalthea-en",
		Age:             "Young Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_ph",
		Characteristics: "Engaging, Natural, Cheerful",
		UseCases:        "Casual chat",
	},
	VoiceAura2Andromeda: {
		ID:              VoiceAura2Andromeda,
		Name:            "Andromeda",
		Model:           "aura-2-andromeda-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Casual, Expressive, Comfortable",
		UseCases:        "Customer service, IVR",
	},
	VoiceAura2Apollo: {
		ID:              VoiceAura2Apollo,
		Name:            "Apollo",
		Model:           "aura-2-apollo-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Confident, Comfortable, Casual",
		UseCases:        "Casual chat",
	},
	VoiceAura2Arcas: {
		ID:              VoiceAura2Arcas,
		Name:            "Arcas",
		Model:           "aura-2-arcas-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Natural, Smooth, Clear, Comfortable",
		UseCases:        "Customer service, casual chat",
	},
	VoiceAura2Aries: {
		ID:              VoiceAura2Aries,
		Name:            "Aries",
		Model:           "aura-2-aries-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Warm, Energetic, Caring",
		UseCases:        "Casual chat",
	},
	VoiceAura2Asteria: {
		ID:              VoiceAura2Asteria,
		Name:            "Asteria",
		Model:           "aura-2-asteria-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Clear, Confident, Knowledgeable, Energetic",
		UseCases:        "Advertising",
	},
	VoiceAura2Athena: {
		ID:              VoiceAura2Athena,
		Name:            "Athena",
		Model:           "aura-2-athena-en",
		Age:             "Mature",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Calm, Smooth, Professional",
		UseCases:        "Storytelling",
	},
	VoiceAura2Atlas: {
		ID:              VoiceAura2Atlas,
		Name:            "Atlas",
		Model:           "aura-2-atlas-en",
		Age:             "Mature",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Enthusiastic, Confident, Approachable, Friendly",
		UseCases:        "Advertising",
	},
	VoiceAura2Aurora: {
		ID:              VoiceAura2Aurora,
		Name:            "Aurora",
		Model:           "aura-2-aurora-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Cheerful, Expressive, Energetic",
		UseCases:        "Interview",
	},
	VoiceAura2Callista: {
		ID:              VoiceAura2Callista,
		Name:            "Callista",
		Model:           "aura-2-callista-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Clear, Energetic, Professional, Smooth",
		UseCases:        "IVR",
	},
	VoiceAura2Cora: {
		ID:              VoiceAura2Cora,
		Name:            "Cora",
		Model:           "aura-2-cora-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Smooth, Melodic, Caring",
		UseCases:        "Storytelling",
	},
	VoiceAura2Cordelia: {
		ID:              VoiceAura2Cordelia,
		Name:            "Cordelia",
		Model:           "aura-2-cordelia-en",
		Age:             "Young Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Approachable, Warm, Polite",
		UseCases:        "Storytelling",
	},
	VoiceAura2Delia: {
		ID:              VoiceAura2Delia,
		Name:            "Delia",
		Model:           "aura-2-delia-en",
		Age:             "Young Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Casual, Friendly, Cheerful, Breathy",
		UseCases:        "Interview ",
	},
	VoiceAura2Draco: {
		ID:              VoiceAura2Draco,
		Name:            "Draco",
		Model:           "aura-2-draco-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_gb",
		Characteristics: "Warm, Approachable, Trustworthy, Baritone",
		UseCases:        "Storytelling",
	},
	VoiceAura2Electra: {
		ID:              VoiceAura2Electra,
		Name:            "Electra",
		Model:           "aura-2-electra-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Professional, Engaging, Knowledgeable",
		UseCases:        "IVR, advertising, customer service",
	},
	VoiceAura2Harmonia: {
		ID:              VoiceAura2Harmonia,
		Name:            "Harmonia",
		Model:           "aura-2-harmonia-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Empathetic, Clear, Calm, Confident",
		UseCases:        "Customer service",
	},
	VoiceAura2Helena: {
		ID:              VoiceAura2Helena,
		Name:            "Helena",
		Model:           "aura-2-helena-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Caring, Natural, Positive, Friendly, Raspy",
		UseCases:        "IVR, casual chat",
	},
	VoiceAura2Hera: {
		ID:              VoiceAura2Hera,
		Name:            "Hera",
		Model:           "aura-2-hera-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Smooth, Warm, Professional",
		UseCases:        "Informative",
	},
	VoiceAura2Hermes: {
		ID:              VoiceAura2Hermes,
		Name:            "Hermes",
		Model:           "aura-2-hermes-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Expressive, Engaging, Professional",
		UseCases:        "Informative",
	},
	VoiceAura2Hyperion: {
		ID:              VoiceAura2Hyperion,
		Name:            "Hyperion",
		Model:           "aura-2-hyperion-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_au",
		Characteristics: "Caring, Warm, Empathetic",
		UseCases:        "Interview",
	},
	VoiceAura2Iris: {
		ID:              VoiceAura2Iris,
		Name:            "Iris",
		Model:           "aura-2-iris-en",
		Age:             "Young Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Cheerful, Positive, Approachable",
		UseCases:        "IVR, advertising, customer service",
	},
	VoiceAura2Janus: {
		ID:              VoiceAura2Janus,
		Name:            "Janus",
		Model:           "aura-2-janus-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Southern, Smooth, Trustworthy",
		UseCases:        "Storytelling",
	},
	VoiceAura2Juno: {
		ID:              VoiceAura2Juno,
		Name:            "Juno",
		Model:           "aura-2-juno-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Natural, Engaging, Melodic, Breathy",
		UseCases:        "Interview",
	},
	VoiceAura2Jupiter: {
		ID:              VoiceAura2Jupiter,
		Name:            "Jupiter",
		Model:           "aura-2-jupiter-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Expressive, Knowledgeable, Baritone",
		UseCases:        "Informative",
	},
	VoiceAura2Luna: {
		ID:              VoiceAura2Luna,
		Name:            "Luna",
		Model:           "aura-2-luna-en",
		Age:             "Young Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Friendly, Natural, Engaging",
		UseCases:        "IVR",
	},
	VoiceAura2Mars: {
		ID:              VoiceAura2Mars,
		Name:            "Mars",
		Model:           "aura-2-mars-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Smooth, Patient, Trustworthy, Baritone",
		UseCases:        "Customer service",
	},
	VoiceAura2Minerva: {
		ID:              VoiceAura2Minerva,
		Name:            "Minerva",
		Model:           "aura-2-minerva-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Positive, Friendly, Natural",
		UseCases:        "Storytelling",
	},
	VoiceAura2Neptune: {
		ID:              VoiceAura2Neptune,
		Name:            "Neptune",
		Model:           "aura-2-neptune-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Professional, Patient, Polite",
		UseCases:        "Customer service",
	},
	VoiceAura2Odysseus: {
		ID:              VoiceAura2Odysseus,
		Name:            "Odysseus",
		Model:           "aura-2-odysseus-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Calm, Smooth, Comfortable, Professional",
		UseCases:        "Advertising",
	},
	VoiceAura2Ophelia: {
		ID:              VoiceAura2Ophelia,
		Name:            "Ophelia",
		Model:           "aura-2-ophelia-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Expressive, Enthusiastic, Cheerful",
		UseCases:        "Interview",
	},
	VoiceAura2Orion: {
		ID:              VoiceAura2Orion,
		Name:            "Orion",
		Model:           "aura-2-orion-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Approachable, Comfortable, Calm, Polite",
		UseCases:        "Informative",
	},
	VoiceAura2Orpheus: {
		ID:              VoiceAura2Orpheus,
		Name:            "Orpheus",
		Model:           "aura-2-orpheus-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Professional, Clear, Confident, Trustworthy",
		UseCases:        "Customer service, storytelling",
	},
	VoiceAura2Pandora: {
		ID:              VoiceAura2Pandora,
		Name:            "Pandora",
		Model:           "aura-2-pandora-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_gb",
		Characteristics: "Smooth, Calm, Melodic, Breathy",
		UseCases:        "IVR, informative",
	},
	VoiceAura2Phoebe: {
		ID:              VoiceAura2Phoebe,
		Name:            "Phoebe",
		Model:           "aura-2-phoebe-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Energetic, Warm, Casual",
		UseCases:        "Customer service",
	},
	VoiceAura2Pluto: {
		ID:              VoiceAura2Pluto,
		Name:            "Pluto",
		Model:           "aura-2-pluto-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Smooth, Calm, Empathetic, Baritone",
		UseCases:        "Interview, storytelling",
	},
	VoiceAura2Saturn: {
		ID:              VoiceAura2Saturn,
		Name:            "Saturn",
		Model:           "aura-2-saturn-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Knowledgeable, Confident, Baritone",
		UseCases:        "Customer service",
	},
	VoiceAura2Selene: {
		ID:              VoiceAura2Selene,
		Name:            "Selene",
		Model:           "aura-2-selene-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Expressive, Engaging, Energetic",
		UseCases:        "Informative",
	},
	VoiceAura2Thalia: {
		ID:              VoiceAura2Thalia,
		Name:            "Thalia",
		Model:           "aura-2-thalia-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Clear, Confident, Energetic, Enthusiastic",
		UseCases:        "Casual chat, customer service, IVR",
	},
	VoiceAura2Theia: {
		ID:              VoiceAura2Theia,
		Name:            "Theia",
		Model:           "aura-2-theia-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_au",
		Characteristics: "Expressive, Polite, Sincere",
		UseCases:        "Informative",
	},
	VoiceAura2Vesta: {
		ID:              VoiceAura2Vesta,
		Name:            "Vesta",
		Model:           "aura-2-vesta-en",
		Age:             "Adult",
		Gender:          "Female",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Natural, Expressive, Patient, Empathetic",
		UseCases:        "Customer service, interview, storytelling",
	},
	VoiceAura2Zeus: {
		ID:              VoiceAura2Zeus,
		Name:            "Zeus",
		Model:           "aura-2-zeus-en",
		Age:             "Adult",
		Gender:          "Male",
		Language:        "English",
		Locale:          "en_us",
		Characteristics: "Deep, Trustworthy, Smooth",
		UseCases:        "IVR",
	},
}
