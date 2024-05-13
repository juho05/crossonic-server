package responses

type SubsonicError int

const (
	SubsonicErrorGeneric                                                  SubsonicError = 0
	SubsonicErrorRequiredParameterMissing                                 SubsonicError = 10
	SubsonicErrorIncompatibleSubsonicRestProtocolVersionClientMustUpgrade SubsonicError = 20
	SubsonicErrorIncompatibleSubsonicRestProtocolVersionServerMustUpgrade SubsonicError = 30
	SubsonicErrorWrongUsernameOrPassword                                  SubsonicError = 40
	SubsonicErrorTokenAuthenticationNotSupported                          SubsonicError = 41
	SubsonicErrorUserNotAuthorized                                        SubsonicError = 50
	SubsonicErrorTrialOver                                                SubsonicError = 60
	SubsonicErrorNotFound                                                 SubsonicError = 70
)
