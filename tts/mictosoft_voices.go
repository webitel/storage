package tts

import (
	"encoding/json"
)

//https://docs.microsoft.com/en-us/azure/cognitive-services/speech-service/language-support?tabs=speechtotext#voice-styles-and-roles
const listVoices = `{
    "af-ZA": {
        "Female": [
            "af-ZA-AdriNeural"
        ],
        "Male": [
            "af-ZA-WillemNeural"
        ]
    },
    "am-ET": {
        "Female": [
            "am-ET-MekdesNeural"
        ],
        "Male": [
            "am-ET-AmehaNeural"
        ]
    },
    "ar-DZ": {
        "Female": [
            "ar-DZ-AminaNeural"
        ],
        "Male": [
            "ar-DZ-IsmaelNeural"
        ]
    },
    "ar-BH": {
        "Female": [
            "ar-BH-LailaNeural"
        ],
        "Male": [
            "ar-BH-AliNeural"
        ]
    },
    "ar-EG": {
        "Female": [
            "ar-EG-SalmaNeural"
        ],
        "Male": [
            "ar-EG-ShakirNeural"
        ]
    },
    "ar-IQ": {
        "Female": [
            "ar-IQ-RanaNeural"
        ],
        "Male": [
            "ar-IQ-BasselNeural"
        ]
    },
    "ar-JO": {
        "Female": [
            "ar-JO-SanaNeural"
        ],
        "Male": [
            "ar-JO-TaimNeural"
        ]
    },
    "ar-KW": {
        "Female": [
            "ar-KW-NouraNeural"
        ],
        "Male": [
            "ar-KW-FahedNeural"
        ]
    },
    "ar-LY": {
        "Female": [
            "ar-LY-ImanNeural"
        ],
        "Male": [
            "ar-LY-OmarNeural"
        ]
    },
    "ar-MA": {
        "Female": [
            "ar-MA-MounaNeural"
        ],
        "Male": [
            "ar-MA-JamalNeural"
        ]
    },
    "ar-QA": {
        "Female": [
            "ar-QA-AmalNeural"
        ],
        "Male": [
            "ar-QA-MoazNeural"
        ]
    },
    "ar-SA": {
        "Female": [
            "ar-SA-ZariyahNeural"
        ],
        "Male": [
            "ar-SA-HamedNeural"
        ]
    },
    "ar-SY": {
        "Female": [
            "ar-SY-AmanyNeural"
        ],
        "Male": [
            "ar-SY-LaithNeural"
        ]
    },
    "ar-TN": {
        "Female": [
            "ar-TN-ReemNeural"
        ],
        "Male": [
            "ar-TN-HediNeural"
        ]
    },
    "ar-AE": {
        "Female": [
            "ar-AE-FatimaNeural"
        ],
        "Male": [
            "ar-AE-HamdanNeural"
        ]
    },
    "ar-YE": {
        "Female": [
            "ar-YE-MaryamNeural"
        ],
        "Male": [
            "ar-YE-SalehNeural"
        ]
    },
    "bn-BD": {
        "Female": [
            "bn-BD-NabanitaNeural"
        ],
        "Male": [
            "bn-BD-PradeepNeural"
        ]
    },
    "bn-IN": {
        "Female": [
            "bn-IN-TanishaaNeural"
        ],
        "Male": [
            "bn-IN-BashkarNeural"
        ]
    },
    "bg-BG": {
        "Female": [
            "bg-BG-KalinaNeural"
        ],
        "Male": [
            "bg-BG-BorislavNeural"
        ]
    },
    "my-MM": {
        "Female": [
            "my-MM-NilarNeural"
        ],
        "Male": [
            "my-MM-ThihaNeural"
        ]
    },
    "ca-ES": {
        "Female": [
            "ca-ES-AlbaNeural",
            "ca-ES-JoanaNeural"
        ],
        "Male": [
            "ca-ES-EnricNeural"
        ]
    },
    "zh-HK": {
        "Female": [
            "zh-HK-HiuGaaiNeural",
            "zh-HK-HiuMaanNeural"
        ],
        "Male": [
            "zh-HK-WanLungNeural"
        ]
    },
    "zh-CN": {
        "Female": [
            "zh-CN-XiaochenNeural",
            "zh-CN-XiaohanNeural",
            "zh-CN-XiaomoNeural",
            "zh-CN-XiaoqiuNeural",
            "zh-CN-XiaoruiNeural",
            "zh-CN-XiaoshuangNeural",
            "zh-CN-XiaoxiaoNeural",
            "zh-CN-XiaoxuanNeural",
            "zh-CN-XiaoyanNeural",
            "zh-CN-XiaoyouNeural"
        ],
        "Male": [
            "zh-CN-YunxiNeural",
            "zh-CN-YunyangNeural",
            "zh-CN-YunyeNeural"
        ]
    },
    "zh-TW": {
        "Female": [
            "zh-TW-HsiaoChenNeural",
            "zh-TW-HsiaoYuNeural"
        ],
        "Male": [
            "zh-TW-YunJheNeural"
        ]
    },
    "hr-HR": {
        "Female": [
            "hr-HR-GabrijelaNeural"
        ],
        "Male": [
            "hr-HR-SreckoNeural"
        ]
    },
    "cs-CZ": {
        "Female": [
            "cs-CZ-VlastaNeural"
        ],
        "Male": [
            "cs-CZ-AntoninNeural"
        ]
    },
    "da-DK": {
        "Female": [
            "da-DK-ChristelNeural"
        ],
        "Male": [
            "da-DK-JeppeNeural"
        ]
    },
    "nl-BE": {
        "Female": [
            "nl-BE-DenaNeural"
        ],
        "Male": [
            "nl-BE-ArnaudNeural"
        ]
    },
    "nl-NL": {
        "Female": [
            "nl-NL-ColetteNeural",
            "nl-NL-FennaNeural"
        ],
        "Male": [
            "nl-NL-MaartenNeural"
        ]
    },
    "en-AU": {
        "Female": [
            "en-AU-NatashaNeural"
        ],
        "Male": [
            "en-AU-WilliamNeural"
        ]
    },
    "en-CA": {
        "Female": [
            "en-CA-ClaraNeural"
        ],
        "Male": [
            "en-CA-LiamNeural"
        ]
    },
    "en-HK": {
        "Female": [
            "en-HK-YanNeural"
        ],
        "Male": [
            "en-HK-SamNeural"
        ]
    },
    "en-IN": {
        "Female": [
            "en-IN-NeerjaNeural"
        ],
        "Male": [
            "en-IN-PrabhatNeural"
        ]
    },
    "en-IE": {
        "Female": [
            "en-IE-EmilyNeural"
        ],
        "Male": [
            "en-IE-ConnorNeural"
        ]
    },
    "en-KE": {
        "Female": [
            "en-KE-AsiliaNeural"
        ],
        "Male": [
            "en-KE-ChilembaNeural"
        ]
    },
    "en-NZ": {
        "Female": [
            "en-NZ-MollyNeural"
        ],
        "Male": [
            "en-NZ-MitchellNeural"
        ]
    },
    "en-NG": {
        "Female": [
            "en-NG-EzinneNeural"
        ],
        "Male": [
            "en-NG-AbeoNeural"
        ]
    },
    "en-PH": {
        "Female": [
            "en-PH-RosaNeural"
        ],
        "Male": [
            "en-PH-JamesNeural"
        ]
    },
    "en-SG": {
        "Female": [
            "en-SG-LunaNeural"
        ],
        "Male": [
            "en-SG-WayneNeural"
        ]
    },
    "en-ZA": {
        "Female": [
            "en-ZA-LeahNeural"
        ],
        "Male": [
            "en-ZA-LukeNeural"
        ]
    },
    "en-TZ": {
        "Female": [
            "en-TZ-ImaniNeural"
        ],
        "Male": [
            "en-TZ-ElimuNeural"
        ]
    },
    "en-GB": {
        "Female": [
            "en-GB-LibbyNeural",
            "en-GB-MiaNeural",
            "en-GB-SoniaNeural"
        ],
        "Male": [
            "en-GB-RyanNeural"
        ]
    },
    "en-US": {
        "Female": [
            "en-US-AmberNeural",
            "en-US-AriaNeural",
            "en-US-AshleyNeural",
            "en-US-CoraNeural",
            "en-US-ElizabethNeural",
            "en-US-JennyNeural",
            "en-US-MichelleNeural",
            "en-US-MonicaNeural",
            "en-US-SaraNeural"
        ],
        "Kid": [
            "en-US-AnaNeural"
        ],
        "Male": [
            "en-US-BrandonNeural",
            "en-US-ChristopherNeural",
            "en-US-EricNeural",
            "en-US-GuyNeural",
            "en-US-JacobNeural"
        ]
    },
    "en-US as the primary default. Additional locales are supported using SSML": {
        "Female": [
            "en-US-JennyMultilingualNeural"
        ]
    },
    "et-EE": {
        "Female": [
            "et-EE-AnuNeural"
        ],
        "Male": [
            "et-EE-KertNeural"
        ]
    },
    "fil-PH": {
        "Female": [
            "fil-PH-BlessicaNeural"
        ],
        "Male": [
            "fil-PH-AngeloNeural"
        ]
    },
    "fi-FI": {
        "Female": [
            "fi-FI-NooraNeural",
            "fi-FI-SelmaNeural"
        ],
        "Male": [
            "fi-FI-HarriNeural"
        ]
    },
    "fr-BE": {
        "Female": [
            "fr-BE-CharlineNeural"
        ],
        "Male": [
            "fr-BE-GerardNeural"
        ]
    },
    "fr-CA": {
        "Female": [
            "fr-CA-SylvieNeural"
        ],
        "Male": [
            "fr-CA-AntoineNeural",
            "fr-CA-JeanNeural"
        ]
    },
    "fr-FR": {
        "Female": [
            "fr-FR-DeniseNeural"
        ],
        "Male": [
            "fr-FR-HenriNeural"
        ]
    },
    "fr-CH": {
        "Female": [
            "fr-CH-ArianeNeural"
        ],
        "Male": [
            "fr-CH-FabriceNeural"
        ]
    },
    "gl-ES": {
        "Female": [
            "gl-ES-SabelaNeural"
        ],
        "Male": [
            "gl-ES-RoiNeural"
        ]
    },
    "de-AT": {
        "Female": [
            "de-AT-IngridNeural"
        ],
        "Male": [
            "de-AT-JonasNeural"
        ]
    },
    "de-DE": {
        "Female": [
            "de-DE-KatjaNeural"
        ],
        "Male": [
            "de-DE-ConradNeural"
        ]
    },
    "de-CH": {
        "Female": [
            "de-CH-LeniNeural"
        ],
        "Male": [
            "de-CH-JanNeural"
        ]
    },
    "el-GR": {
        "Female": [
            "el-GR-AthinaNeural"
        ],
        "Male": [
            "el-GR-NestorasNeural"
        ]
    },
    "gu-IN": {
        "Female": [
            "gu-IN-DhwaniNeural"
        ],
        "Male": [
            "gu-IN-NiranjanNeural"
        ]
    },
    "he-IL": {
        "Female": [
            "he-IL-HilaNeural"
        ],
        "Male": [
            "he-IL-AvriNeural"
        ]
    },
    "hi-IN": {
        "Female": [
            "hi-IN-SwaraNeural"
        ],
        "Male": [
            "hi-IN-MadhurNeural"
        ]
    },
    "hu-HU": {
        "Female": [
            "hu-HU-NoemiNeural"
        ],
        "Male": [
            "hu-HU-TamasNeural"
        ]
    },
    "is-IS": {
        "Female": [
            "is-IS-GudrunNeural"
        ],
        "Male": [
            "is-IS-GunnarNeural"
        ]
    },
    "id-ID": {
        "Female": [
            "id-ID-GadisNeural"
        ],
        "Male": [
            "id-ID-ArdiNeural"
        ]
    },
    "ga-IE": {
        "Female": [
            "ga-IE-OrlaNeural"
        ],
        "Male": [
            "ga-IE-ColmNeural"
        ]
    },
    "it-IT": {
        "Female": [
            "it-IT-ElsaNeural",
            "it-IT-IsabellaNeural"
        ],
        "Male": [
            "it-IT-DiegoNeural"
        ]
    },
    "ja-JP": {
        "Female": [
            "ja-JP-NanamiNeural"
        ],
        "Male": [
            "ja-JP-KeitaNeural"
        ]
    },
    "jv-ID": {
        "Female": [
            "jv-ID-SitiNeural"
        ],
        "Male": [
            "jv-ID-DimasNeural"
        ]
    },
    "kn-IN": {
        "Female": [
            "kn-IN-SapnaNeural"
        ],
        "Male": [
            "kn-IN-GaganNeural"
        ]
    },
    "kk-KZ": {
        "Female": [
            "kk-KZ-AigulNeural"
        ],
        "Male": [
            "kk-KZ-DauletNeural"
        ]
    },
    "km-KH": {
        "Female": [
            "km-KH-SreymomNeural"
        ],
        "Male": [
            "km-KH-PisethNeural"
        ]
    },
    "ko-KR": {
        "Female": [
            "ko-KR-SunHiNeural"
        ],
        "Male": [
            "ko-KR-InJoonNeural"
        ]
    },
    "lo-LA": {
        "Female": [
            "lo-LA-KeomanyNeural"
        ],
        "Male": [
            "lo-LA-ChanthavongNeural"
        ]
    },
    "lv-LV": {
        "Female": [
            "lv-LV-EveritaNeural"
        ],
        "Male": [
            "lv-LV-NilsNeural"
        ]
    },
    "lt-LT": {
        "Female": [
            "lt-LT-OnaNeural"
        ],
        "Male": [
            "lt-LT-LeonasNeural"
        ]
    },
    "mk-MK": {
        "Female": [
            "mk-MK-MarijaNeural"
        ],
        "Male": [
            "mk-MK-AleksandarNeural"
        ]
    },
    "ms-MY": {
        "Female": [
            "ms-MY-YasminNeural"
        ],
        "Male": [
            "ms-MY-OsmanNeural"
        ]
    },
    "ml-IN": {
        "Female": [
            "ml-IN-SobhanaNeural"
        ],
        "Male": [
            "ml-IN-MidhunNeural"
        ]
    },
    "mt-MT": {
        "Female": [
            "mt-MT-GraceNeural"
        ],
        "Male": [
            "mt-MT-JosephNeural"
        ]
    },
    "mr-IN": {
        "Female": [
            "mr-IN-AarohiNeural"
        ],
        "Male": [
            "mr-IN-ManoharNeural"
        ]
    },
    "nb-NO": {
        "Female": [
            "nb-NO-IselinNeural",
            "nb-NO-PernilleNeural"
        ],
        "Male": [
            "nb-NO-FinnNeural"
        ]
    },
    "ps-AF": {
        "Female": [
            "ps-AF-LatifaNeural"
        ],
        "Male": [
            "ps-AF-GulNawazNeural"
        ]
    },
    "fa-IR": {
        "Female": [
            "fa-IR-DilaraNeural"
        ],
        "Male": [
            "fa-IR-FaridNeural"
        ]
    },
    "pl-PL": {
        "Female": [
            "pl-PL-AgnieszkaNeural",
            "pl-PL-ZofiaNeural"
        ],
        "Male": [
            "pl-PL-MarekNeural"
        ]
    },
    "pt-BR": {
        "Female": [
            "pt-BR-FranciscaNeural"
        ],
        "Male": [
            "pt-BR-AntonioNeural"
        ]
    },
    "pt-PT": {
        "Female": [
            "pt-PT-FernandaNeural",
            "pt-PT-RaquelNeural"
        ],
        "Male": [
            "pt-PT-DuarteNeural"
        ]
    },
    "ro-RO": {
        "Female": [
            "ro-RO-AlinaNeural"
        ],
        "Male": [
            "ro-RO-EmilNeural"
        ]
    },
    "ru-RU": {
        "Female": [
            "ru-RU-DariyaNeural",
            "ru-RU-SvetlanaNeural"
        ],
        "Male": [
            "ru-RU-DmitryNeural"
        ]
    },
    "sr-RS": {
        "Female": [
            "sr-RS-SophieNeural"
        ],
        "Male": [
            "sr-RS-NicholasNeural"
        ]
    },
    "si-LK": {
        "Female": [
            "si-LK-ThiliniNeural"
        ],
        "Male": [
            "si-LK-SameeraNeural"
        ]
    },
    "sk-SK": {
        "Female": [
            "sk-SK-ViktoriaNeural"
        ],
        "Male": [
            "sk-SK-LukasNeural"
        ]
    },
    "sl-SI": {
        "Female": [
            "sl-SI-PetraNeural"
        ],
        "Male": [
            "sl-SI-RokNeural"
        ]
    },
    "so-SO": {
        "Female": [
            "so-SO-UbaxNeural"
        ],
        "Male": [
            "so-SO-MuuseNeural"
        ]
    },
    "es-AR": {
        "Female": [
            "es-AR-ElenaNeural"
        ],
        "Male": [
            "es-AR-TomasNeural"
        ]
    },
    "es-BO": {
        "Female": [
            "es-BO-SofiaNeural"
        ],
        "Male": [
            "es-BO-MarceloNeural"
        ]
    },
    "es-CL": {
        "Female": [
            "es-CL-CatalinaNeural"
        ],
        "Male": [
            "es-CL-LorenzoNeural"
        ]
    },
    "es-CO": {
        "Female": [
            "es-CO-SalomeNeural"
        ],
        "Male": [
            "es-CO-GonzaloNeural"
        ]
    },
    "es-CR": {
        "Female": [
            "es-CR-MariaNeural"
        ],
        "Male": [
            "es-CR-JuanNeural"
        ]
    },
    "es-CU": {
        "Female": [
            "es-CU-BelkysNeural"
        ],
        "Male": [
            "es-CU-ManuelNeural"
        ]
    },
    "es-DO": {
        "Female": [
            "es-DO-RamonaNeural"
        ],
        "Male": [
            "es-DO-EmilioNeural"
        ]
    },
    "es-EC": {
        "Female": [
            "es-EC-AndreaNeural"
        ],
        "Male": [
            "es-EC-LuisNeural"
        ]
    },
    "es-SV": {
        "Female": [
            "es-SV-LorenaNeural"
        ],
        "Male": [
            "es-SV-RodrigoNeural"
        ]
    },
    "es-GQ": {
        "Female": [
            "es-GQ-TeresaNeural"
        ],
        "Male": [
            "es-GQ-JavierNeural"
        ]
    },
    "es-GT": {
        "Female": [
            "es-GT-MartaNeural"
        ],
        "Male": [
            "es-GT-AndresNeural"
        ]
    },
    "es-HN": {
        "Female": [
            "es-HN-KarlaNeural"
        ],
        "Male": [
            "es-HN-CarlosNeural"
        ]
    },
    "es-MX": {
        "Female": [
            "es-MX-DaliaNeural"
        ],
        "Male": [
            "es-MX-JorgeNeural"
        ]
    },
    "es-NI": {
        "Female": [
            "es-NI-YolandaNeural"
        ],
        "Male": [
            "es-NI-FedericoNeural"
        ]
    },
    "es-PA": {
        "Female": [
            "es-PA-MargaritaNeural"
        ],
        "Male": [
            "es-PA-RobertoNeural"
        ]
    },
    "es-PY": {
        "Female": [
            "es-PY-TaniaNeural"
        ],
        "Male": [
            "es-PY-MarioNeural"
        ]
    },
    "es-PE": {
        "Female": [
            "es-PE-CamilaNeural"
        ],
        "Male": [
            "es-PE-AlexNeural"
        ]
    },
    "es-PR": {
        "Female": [
            "es-PR-KarinaNeural"
        ],
        "Male": [
            "es-PR-VictorNeural"
        ]
    },
    "es-ES": {
        "Female": [
            "es-ES-ElviraNeural"
        ],
        "Male": [
            "es-ES-AlvaroNeural"
        ]
    },
    "es-UY": {
        "Female": [
            "es-UY-ValentinaNeural"
        ],
        "Male": [
            "es-UY-MateoNeural"
        ]
    },
    "es-US": {
        "Female": [
            "es-US-PalomaNeural"
        ],
        "Male": [
            "es-US-AlonsoNeural"
        ]
    },
    "es-VE": {
        "Female": [
            "es-VE-PaolaNeural"
        ],
        "Male": [
            "es-VE-SebastianNeural"
        ]
    },
    "su-ID": {
        "Female": [
            "su-ID-TutiNeural"
        ],
        "Male": [
            "su-ID-JajangNeural"
        ]
    },
    "sw-KE": {
        "Female": [
            "sw-KE-ZuriNeural"
        ],
        "Male": [
            "sw-KE-RafikiNeural"
        ]
    },
    "sw-TZ": {
        "Female": [
            "sw-TZ-RehemaNeural"
        ],
        "Male": [
            "sw-TZ-DaudiNeural"
        ]
    },
    "sv-SE": {
        "Female": [
            "sv-SE-HilleviNeural",
            "sv-SE-SofieNeural"
        ],
        "Male": [
            "sv-SE-MattiasNeural"
        ]
    },
    "ta-IN": {
        "Female": [
            "ta-IN-PallaviNeural"
        ],
        "Male": [
            "ta-IN-ValluvarNeural"
        ]
    },
    "ta-SG": {
        "Female": [
            "ta-SG-VenbaNeural"
        ],
        "Male": [
            "ta-SG-AnbuNeural"
        ]
    },
    "ta-LK": {
        "Female": [
            "ta-LK-SaranyaNeural"
        ],
        "Male": [
            "ta-LK-KumarNeural"
        ]
    },
    "te-IN": {
        "Female": [
            "te-IN-ShrutiNeural"
        ],
        "Male": [
            "te-IN-MohanNeural"
        ]
    },
    "th-TH": {
        "Female": [
            "th-TH-AcharaNeural",
            "th-TH-PremwadeeNeural"
        ],
        "Male": [
            "th-TH-NiwatNeural"
        ]
    },
    "tr-TR": {
        "Female": [
            "tr-TR-EmelNeural"
        ],
        "Male": [
            "tr-TR-AhmetNeural"
        ]
    },
    "uk-UA": {
        "Female": [
            "uk-UA-PolinaNeural"
        ],
        "Male": [
            "uk-UA-OstapNeural"
        ]
    },
    "ur-IN": {
        "Female": [
            "ur-IN-GulNeural"
        ],
        "Male": [
            "ur-IN-SalmanNeural"
        ]
    },
    "ur-PK": {
        "Female": [
            "ur-PK-UzmaNeural"
        ],
        "Male": [
            "ur-PK-AsadNeural"
        ]
    },
    "uz-UZ": {
        "Female": [
            "uz-UZ-MadinaNeural"
        ],
        "Male": [
            "uz-UZ-SardorNeural"
        ]
    },
    "vi-VN": {
        "Female": [
            "vi-VN-HoaiMyNeural"
        ],
        "Male": [
            "vi-VN-NamMinhNeural"
        ]
    },
    "cy-GB": {
        "Female": [
            "cy-GB-NiaNeural"
        ],
        "Male": [
            "cy-GB-AledNeural"
        ]
    },
    "zu-ZA": {
        "Female": [
            "zu-ZA-ThandoNeural"
        ],
        "Male": [
            "zu-ZA-ThembaNeural"
        ]
    },
    "az-AZ": {
        "Female": [
            "az-AZ-BanuNeural"
        ],
        "Male": [
            "az-AZ-BabekNeural"
        ]
    }
}`

var parsedListVoice map[string]map[string][]string

func init() {
	json.Unmarshal([]byte(listVoices), &parsedListVoice)
}

func microsoftLocalesNameMapping(locale, gender string) string {
	if l, ok := parsedListVoice[locale]; ok {
		if v, ok := l[gender]; ok && len(v) > 0 {
			return v[0]
		}
	}

	return locale
}
