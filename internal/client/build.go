package client

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func RuntimeDirectory(home, directory string) (string, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	digest := md5.Sum([]byte(filepath.ToSlash(resolved)))
	return filepath.Join(home, "AppData", "LocalLow", "netease", "Marvel MM CN", hex.EncodeToString(digest[:])), nil
}

type BuildProfile struct {
	BaseVersion                  string
	Branch                       string
	ContentVersion               string
	ContentList                  string
	ContentScriptsSHA256         string
	DLLSHA256                    string
	AssemblySHA256               string
	ScriptsSHA256                string
	BattleSHA256                 string
	LoginSHA256                  string
	ServerClientSHA256           string
	FriendTipSHA256              string
	FriendInfoSHA256             string
	FriendCompSHA256             string
	BattleCompSHA256             string
	BattleStageSHA256            string
	SceneTimeSHA256              string
	PaymentSHA256                string
	PatchUtilsSHA256             string
	PatchStatusSHA256            string
	LogicSHA256                  string
	AIEngineSHA256               string
	BattlePanelSHA256            string
	WindowSHA256                 string
	SkipGuideSHA256              string
	HeroHighlightSHA256          string
	DrawSHA256                   string
	DrawUpSHA256                 string
	DrawUpInvincibleSHA256       string
	DrawAnimatorSHA256           string
	DrawAnimatorInvincibleSHA256 string
	NovelSHA256                  string
	ExpSpriteSHA256              string
	GallerySHA256                string
	ManualSHA256                 string
	UniMatrixHelperSHA256        string
	RandomNameSHA256             string
	NumFormatSHA256              string
	PatchedScriptsSHA256         string
}

var Supported = BuildProfile{
	BaseVersion:                  "2.1.3",
	Branch:                       "releasecn",
	ContentVersion:               "2.1.20",
	ContentList:                  "https://g124.update.netease.com/pl/patchlist_win32_releasecn",
	ContentScriptsSHA256:         "3c85d0bbd794bfedb350098e77ac6afad3395a0036f76dfc694246b302997c69",
	DLLSHA256:                    "92a283058c9a857cdd786755039b127e5858a6de33f01dca3eb25a177044d516",
	AssemblySHA256:               "7dfd282cbc3e9e71b0332fe19b4b2c0e17a75c5faa1d3e47ca8fad951f15b53b",
	ScriptsSHA256:                "32dc0ab3c2fd1cf4424909b7ac0f400814727f19e5c8639e7b83fffec45ecc67",
	BattleSHA256:                 "c9e2ddb58832b1150b3c6b4049cfaff1c97587a630e9ad21572706ead696f628",
	LoginSHA256:                  "5e82b5ca2a6fee0bee4dac9f35d67fabd0182dd25636fc5baa242e4e58b20505",
	ServerClientSHA256:           "da97ca7327cd75581d683816bae29119181c278f35baba6f91f2549fd82f7e41",
	FriendTipSHA256:              "238ffca33b0b1867b1864a0cee4a7a26661a29fa03b3b1bdc737724f3dd39e6c",
	FriendInfoSHA256:             "88b0996757f4193a2da17f00595625e47ee134b0da9709e78a1b1d9e87ede308",
	FriendCompSHA256:             "21753ce1886aa7758c38c4045aa5e25a5c196b50949771320a43f5ad0eb18a23",
	BattleCompSHA256:             "2523369801a0762b23d0f576970202cc1481cd9e2faf8225e2d9839215c120cd",
	BattleStageSHA256:            "601210a0fadb5566a7fd1ff04e308d3d6847940984e41c6e13040a88f20688b6",
	SceneTimeSHA256:              "a974da6bfb9eae38e46effa81f7b0a6e0580e65d7813b17b4c6296d77ae6ce41",
	PaymentSHA256:                "1279497f4a4bdb3764d8678d5e9a4969489d2d1a9b79cbe623993d60749f7af7",
	PatchUtilsSHA256:             "609c5c1d968a730ccccdbfaf3d29828b7e7d999c145f194d09e04cd8be5e33cd",
	PatchStatusSHA256:            "d9b8ad79628f66ddd6bb8a8709bdb9051ba6f30c0da99582b4ee27ca5d939cd4",
	LogicSHA256:                  "3bcb098a3d612353703f507e32960fa015f13579790fd863199ff18069004bf7",
	AIEngineSHA256:               "62b105fdb25b056f63a995678b2c7f717f558b6ddbc4d1720cfa1cc424d68b9c",
	BattlePanelSHA256:            "3bb754d4c2256147bc6047cfc66a3141ec89104248c84840fe0655969c7ae124",
	WindowSHA256:                 "16bdc7a63a6ca09a168c794bea49dc299006da613f9bcfd1f1bf12a71a05734e",
	SkipGuideSHA256:              "cf31d72de97e23b0fb8576d17b1ba80f604332dfdfac663ddff291a79d82a6b8",
	HeroHighlightSHA256:          "123afecf95b7e2ba166489547f495105e352b83dc34846f87b089742c7c5e427",
	DrawSHA256:                   "a93f6255626dc900c24c24d2ca53323b4ea2a073306928bb03652903e5712a1a",
	DrawUpSHA256:                 "c64c50067931e6705105aaf2712affe4171e516aca09804f9e61824f55f32097",
	DrawUpInvincibleSHA256:       "e059658447d2c022aa4eb36cdc3a9933286345479ed293676f538e60add9a1fa",
	DrawAnimatorSHA256:           "3347bb2484e6a40ef5fd51a213450b247d830b98edbab96834d88276ccf3b27c",
	DrawAnimatorInvincibleSHA256: "267b26d9a5fa91d70a34a00415513db45311b9a07e99172e7cdb7f4e5f3dceba",
	NovelSHA256:                  "5dac569e22c4b3a68af466469192f65cfbb1d3c98bad03901ce888d1ff57519c",
	ExpSpriteSHA256:              "66ff844ae1de3784e653a6706c64c7643fce70599d387a082a6354e797c9a11a",
	GallerySHA256:                "0769e5dbc6744edfd357f281ced834a1af736b5adada21733a0b54eb1be8c6b6",
	ManualSHA256:                 "665878fdc59d431470671d1bacb4aa01e171cf33dd112f307a2c001118da2e4d",
	UniMatrixHelperSHA256:        "6320379d9a9fa12f1f65b07760f60ef96e86fa590e57a8b06fdc04c3e96ca667",
	RandomNameSHA256:             "372f986d47291f040010501c535c3b28ac2655451eacc96636d90056d8498a77",
	NumFormatSHA256:              "4afa806ae6c2273b7319f8aa752c28e19fc790c2e8ca7d977b5ed11b30b7f885",
	PatchedScriptsSHA256:         "d182d0f3468eb69a3248ace0fdc88fa8fd8e7484f5ca985dac59f4441e64ecc1",
}

const PatchListPath = "/native/patchlist_win32.json"

func (p BuildProfile) ContentMarker() string {
	return p.Branch + "/" + p.ContentVersion + "/" + p.ContentScriptsSHA256
}

func DetectBuild(directory string) (BuildProfile, error) {
	assemblyHash, err := hashFile(filepath.Join(directory, "GameAssembly.dll"))
	if err != nil {
		return BuildProfile{}, err
	}
	scriptsHash, err := hashFile(filepath.Join(directory, "Marvel MM CN_Data", "StreamingAssets", "scripts.pck"))
	if err != nil {
		return BuildProfile{}, err
	}
	if Supported.AssemblySHA256 != assemblyHash || Supported.ScriptsSHA256 != scriptsHash {
		return BuildProfile{}, errors.New("unsupported or modified client build")
	}
	return Supported, nil
}

func profileForDLL(data []byte) (BuildProfile, error) {
	if hash(data) != Supported.DLLSHA256 {
		return BuildProfile{}, errors.New("unsupported client DLL")
	}
	return Supported, nil
}

func hash(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func Digest(value []byte) string {
	return hash(value)
}

func hashFile(name string) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
