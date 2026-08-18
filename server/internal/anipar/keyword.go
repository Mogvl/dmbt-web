package anipar

// MARK: audio / video keyword tables

type audioInfoValue struct {
	codec      string
	channels   string
	trackCount int
}

var audioChannels = map[string]bool{
	"2.0CH": true,
	"2CH":   true,
	"5.1":   true,
	"5.1CH": true,
}

var audioCompoundTerms = map[string]audioInfoValue{
	"DTS5.1":    {codec: "DTS", channels: "5.1"},
	"TRUEHD5.1": {codec: "TRUEHD", channels: "5.1"},
	"AACX2":     {codec: "AAC", trackCount: 2},
	"AAC×2":     {codec: "AAC", trackCount: 2},
	"AACX3":     {codec: "AAC", trackCount: 3},
	"AAC×3":     {codec: "AAC", trackCount: 3},
	"AACX4":     {codec: "AAC", trackCount: 4},
	"AAC×4":     {codec: "AAC", trackCount: 4},
	"FLACX2":    {codec: "FLAC", trackCount: 2},
	"FLAC×2":    {codec: "FLAC", trackCount: 2},
	"FLACX3":    {codec: "FLAC", trackCount: 3},
	"FLAC×3":    {codec: "FLAC", trackCount: 3},
	"FLACX4":    {codec: "FLAC", trackCount: 4},
	"FLAC×4":    {codec: "FLAC", trackCount: 4},
}

var audioCodecs = map[string]bool{
	"DTS":      true,
	"DTS-ES":   true,
	"EAC3&AAC": true,
	"AAC":      true,
	"QAAC":     true,
	"AC3":      true,
	"EAC3":     true,
	"E-AC-3":   true,
	"FLAC":     true,
	"FLAC/AC3": true,
	"LOSSLESS": true,
	"MP3":      true,
	"WAV":      true,
	"OGG":      true,
	"VORBIS":   true,
	"OPUS":     true,
}

var audioLanguages = map[string]bool{
	"DUALAUDIO":  true,
	"DUAL AUDIO": true,
}

var videoBitDepths = map[string]bool{
	"8BIT":    true,
	"8-BIT":   true,
	"10BIT":   true,
	"10BITS":  true,
	"10-BIT":  true,
	"10-BITS": true,
}

var videoCodecs = map[string]bool{
	"HI10":    true,
	"HI10P":   true,
	"HI444":   true,
	"HI444P":  true,
	"HI444PP": true,
	"H264":    true,
	"H265":    true,
	"H.264":   true,
	"H.265":   true,
	"X264":    true,
	"X265":    true,
	"X.264":   true,
	"AVC":     true,
	"HEVC":    true,
	"HEVC2":   true,
	"DIVX":    true,
	"DIVX5":   true,
	"DIVX6":   true,
	"XVID":    true,
}

var videoFormats = map[string]bool{
	"AVI":  true,
	"RMVB": true,
	"WMV":  true,
	"WMV3": true,
	"WMV9": true,
}

var videoQualities = map[string]bool{
	"HDR": true,
	"HQ":  true,
	"LQ":  true,
}

var videoResolutionTerms = map[string]bool{
	"HD": true,
	"SD": true,
}

type videoInfoValue struct {
	codec      string
	bitDepth   string
	resolution string
	audioCodec string
}

var videoCompoundTerms = map[string]videoInfoValue{
	"AVC-8BIT":         {codec: "AVC", bitDepth: "8BIT"},
	"HEVC_OPUS":        {codec: "HEVC", audioCodec: "OPUS"},
	"HEVC-10BIT":       {codec: "HEVC", bitDepth: "10BIT"},
	"HEVC-10BIT-1440P": {codec: "HEVC", bitDepth: "10BIT", resolution: "1440P"},
	"HEVC-10BIT-2160P": {codec: "HEVC", bitDepth: "10BIT", resolution: "2160P"},
	"HEVC_10BIT":       {codec: "HEVC", bitDepth: "10BIT"},
	"HEVC-8BIT":        {codec: "HEVC", bitDepth: "8BIT"},
	"HEVC_8BIT":        {codec: "HEVC", bitDepth: "8BIT"},
}

var videoFrameRates = map[string]bool{
	"23.976FPS": true,
	"24FPS":     true,
	"29.97FPS":  true,
	"30FPS":     true,
	"60FPS":     true,
	"120FPS":    true,
}

var videoResolutions = map[string]bool{
	"2K": true,
	"4K": true,
}

var sourceSet = map[string]bool{
	"BD":        true,
	"BDRIP":     true,
	"BLURAY":    true,
	"BLU-RAY":   true,
	"BDREMUX":   true,
	"UHDBDRIP":  true,
	"DVD":       true,
	"DVD5":      true,
	"DVD9":      true,
	"DVD-R2J":   true,
	"DVDRIP":    true,
	"DVD-RIP":   true,
	"R2DVD":     true,
	"R2J":       true,
	"R2JDVD":    true,
	"R2JDVDRIP": true,
	"HDTV":      true,
	"HDTVRIP":   true,
	"TVRIP":     true,
	"TV-RIP":    true,
	"WEB":       true,
	"WEBCAST":   true,
	"WEBDL":     true,
	"WEB-DL":    true,
	"WEBRIP":    true,
	"WEB-RIP":   true,
	"WEB-MKV":   true,
	"MASTERRIP": true,
	"DISC1":     true,
	"DISC2":     true,
	"DISC3":     true,
	"DISC4":     true,
	"DISC5":     true,
	"DISC6":     true,
	"DISC7":     true,
	"DISC8":     true,
	"DISC9":     true,
}

// MARK: 语言,字幕等

var platforms = map[string]bool{
	"Baha":     true,
	"Bili":     true,
	"Bilibili": true,
	"BiliBili": true,
	"B-Global": true,
	"ABEMA":    true,
	"CR":       true,
	"AT-X":     true,
	"AT-X版":    true,
	"ViuTV":    true,
	"AMZN":     true,
	"ADN":      true,
	"Sentai":   true,
	"Netflix":  true,
	"NF":       true,
}

var variantsSet = map[string]bool{
	"日配版":            true,
	"中配版":            true,
	"日文配音":           true,
	"中文配音":           true,
	"Chinese Audio":  true,
	"Japanese Audio": true,
	"JPN Audio":      true,
	"Japanese Dub":   true,
	"JP Dub":         true,
	"English Audio":  true,
	"English Dub":    true,
}

var subtitleFormatTerms = map[string]string{
	"ASS":       "ASS",
	"ASSX2":     "ASSx2",
	"ASSX3":     "ASSx3",
	"ASSX4":     "ASSx4",
	"HARDSUB":   "HARDSUB",
	"HARDSUBS":  "HARDSUB",
	"SOFTSUB":   "SOFTSUB",
	"SOFTSUBS":  "SOFTSUB",
	"SUB":       "SUB",
	"SUBBED":    "SUB",
	"SUBTITLED": "SUB",
	"SRT":       "SRT",
	"SRTX2":     "SRTx2",
	"SRTX3":     "SRTx3",
	"SRTX4":     "SRTx4",
}

type subtitleEncodingValue struct {
	format    string
	encoding  string
	encodings []string
}

var subtitleEncodingTerms = map[string]subtitleEncodingValue{
	"GB&BIG5":   {encodings: []string{"GB", "BIG5"}},
	"BIG5&GB":   {encodings: []string{"BIG5", "GB"}},
	"外挂GB/BIG5": {format: "外挂", encodings: []string{"GB", "BIG5"}},
	"GB/BIG5":   {encodings: []string{"GB", "BIG5"}},
	"GB":        {encoding: "GB"},
	"BIG5":      {encoding: "BIG5"},
}

var platformLanguageTerms = map[string][2]string{
	"ViuTV粵語": {"ViuTV", "粵語"},
	"TVB粵語":   {"TVB", "粵語"},
}

type languageSubtitleFormatValue struct {
	language string
	format   string
}

var languageSubtitleFormatTerms = map[string]languageSubtitleFormatValue{
	"代理商粵語":         {language: "粵語"},
	"粵日雙語+內封繁體中文字幕": {language: "繁體中文", format: "內封字幕"},
	"粵語+無對白字幕":      {language: "粵語+無對白"},
}

var subtitleLanguageTerms = map[string]string{
	"CN":        "CN",
	"CHS":       "CHS",
	"CHT":       "CHT",
	"YUE":       "YUE",
	"JPN":       "JPN",
	"JP":        "JP",
	"简体":        "简体",
	"简/繁·日":     "简/繁·日",
	"繁/體":       "繁/體",
	"简繁":        "简繁",
	"国语中字":      "国语中字",
	"繁體":        "繁體",
	"中日双语":      "中日双语",
	"繁日双语":      "繁日双语",
	"简日双语":      "简日双语",
	"繁日雙語":      "繁日雙語",
	"HOY粵語":     "HOY粵語",
	"外挂CHS/CHT": "CHS/CHT",
	"外挂繁简日字幕":   "繁简日",
}

var subtitleLanguagePrefixes = []string{
	"简繁日双语",
	"简繁日语",
	"简繁英日",
	"简繁日英",
	"简繁日",
	"简日双语",
	"简/繁",
	"简繁英",
	"简繁泰",
	"繁简日",
	"中日英",
	"简日",
	"简繁",
	"簡繁",
	"简英",
	"繁體",
	"繁体",
	"繁日",
	"繁英",
	"英文",
	"简体",
	"简",
	"繁",
	"英",
}

// SubtitleFormatSuffixTerms values; "" marks undefined in the original map.
var subtitleFormatSuffixTerms = map[string]string{
	"内嵌字幕": "内嵌字幕",
	"内嵌":   "内嵌",
	"內嵌":   "內嵌",
	"内封字幕": "内封字幕",
	"内封":   "内封",
	"內封":   "內封",
	"外挂字幕": "外挂字幕",
	"外挂":   "外挂",
	"外掛":   "外掛",
	"内挂":   "内挂",
	"字幕":   "",
}

// MARK: 其他标签

var otherTags = map[string]bool{
	"RAW":            true,
	"DUB":            true,
	"DUBBED":         true,
	"retake":         true,
	"SNS":            true,
	"全歌曲特效":          true,
	"无水印":            true,
	"含副音轨":           true,
	"特典":             true,
	"LIVE纯享":         true,
	"无损重制":           true,
	"广播剧_Dream☆Arch": true,
	"国漫":             true,
	"Donghua":        true,
	"特別版":            true,
	"先行版":            true,
	"先行版本":           true,
	"正片先行版":          true,
	"正式版":            true,
	"正式版本":           true,
	"放送版":            true,
	"修订版":            true,
	"修訂版":            true,
	"On-air version": true,
	"年齡限制版":          true,
	"Ani-One":        true,
	"僅限港澳台":          true,
	"僅限港澳台地區":        true,
	"僅限港澳臺地區":        true,
	"仅限港澳台":          true,
	"仅限港澳台地区":        true,
	"重播":             true,
	"End":            true,
	"END":            true,
	"TV + Movie Fin": true,
	"FIN":            true,
	"Fin":            true,
}

// Prefix
var searchPrefix = []string{
	"检索:",
	"检索：",
	"檢索:",
	"檢索：",
	"检索用:",
	"检索用：",
	"檢索用:",
	"檢索用：",
}

var hiringPrefix = []string{"招募", "急募", "字幕社招人", "字幕社招人"}

var otherPrefix = []string{"▶"}

var ignores = map[string]bool{
	"务必查看bt站简介": true,
	"请看bt站简介":   true,
	"添加日语":      true,
	"添加日語":      true,
}
