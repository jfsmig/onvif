# ONVIF Profile S -- operations a CLIENT may issue.
#
# Curated by hand from docs/ONVIF-Profile-S-Specification.pdf (v1.3, November 2019),
# from the "Function List for Clients" table of each feature (sections *.4). The client
# column is the one that matters here: this is a client library, and the two differ --
# 7.11 Media Profile Configuration is Device MANDATORY but Client CONDITIONAL.
#
# Records:
#   service <go-package> <M|C>                    M services gate NewProfileS()
#   feature <section> <title>                     groups the operations below it
#   <go-package> <Operation> <M|C|O|M*> <section>
#
# Field 1 is the Go package directory, which is what CallMethod routes on: `event`, not
# `events`. Every operation is checked against <package>/calls.txt at generation time.
#
# Two kinds of row in the specification's tables are deliberately absent:
#   - Notify, TopicFilter, MessageContentFilter (7.7): not client-callable operations.
#     Notify is the NotificationConsumer side, which a client serves rather than calls.
#   - every row whose Service column is "Core" or "Streaming", wherever it occurs: those are
#     WS-Security, WS-Discovery and RTSP obligations rather than SOAP operations, and nothing
#     here could route them. They appear in 7.1, 7.3, 7.8, 7.9, 8.1 and 8.2 today, and the
#     rule is stated rather than the list enumerated, so that a row in a section added later
#     cannot be dropped silently. Note 7.1 makes HTTP Digest Client MANDATORY, which this
#     library does not yet implement.
# One entry is renamed: 7.5 lists "Reboot"; the WSDL operation is SystemReboot.
# One section title is corrected: 8.17 is "IP Address Filtering" in the body, while its
#   own function-list headers repeat "Relay Outputs" from 8.13. The body title is used.
# M* marks 7.7's footnote: a client shall implement one or both of PullPoint and
#   Base-Notification, so neither group is mandatory on its own.

service device  M   # 7.2 Capabilities is Client MANDATORY
service media   M   # 7.8 Video Streaming is Client MANDATORY
service ptz     C   # 8.3 PTZ (if supported)
service event   C   # 7.7 Event Handling is Client Conditional

feature 7.2   Capabilities
device  GetCapabilities                            M   7.2
device  GetWsdlUrl                                 O   7.2

feature 7.3   Discovery
device  AddScopes                                  O   7.3
device  GetDiscoveryMode                           O   7.3
device  GetScopes                                  O   7.3
device  RemoveScopes                               O   7.3
device  SetDiscoveryMode                           O   7.3
device  SetScopes                                  O   7.3

feature 7.4   Network Configuration
device  GetDNS                                     O   7.4
device  GetHostname                                O   7.4
device  GetNetworkDefaultGateway                   M   7.4
device  GetNetworkInterfaces                       M   7.4
device  GetNetworkProtocols                        O   7.4
device  SetDNS                                     O   7.4
device  SetHostname                                O   7.4
device  SetNetworkDefaultGateway                   M   7.4
device  SetNetworkInterfaces                       M   7.4
device  SetNetworkProtocols                        O   7.4

feature 7.5   System
device  GetDeviceInformation                       M   7.5
device  GetSystemDateAndTime                       O   7.5
device  SetSystemDateAndTime                       O   7.5
device  SetSystemFactoryDefault                    O   7.5
device  SystemReboot                               O   7.5

feature 7.6   User Handling
device  CreateUsers                                M   7.6
device  DeleteUsers                                M   7.6
device  GetUsers                                   M   7.6
device  SetUser                                    M   7.6

feature 7.7   Event Handling
event   CreatePullPointSubscription                M*  7.7
event   GetEventProperties                         O   7.7
event   PullMessages                               M*  7.7
event   Renew                                      M*  7.7
event   SetSynchronizationPoint                    O   7.7
event   Subscribe                                  M*  7.7
event   Unsubscribe                                O   7.7

feature 7.8   Video Streaming
media   GetProfiles                                M   7.8
media   GetStreamUri                               M   7.8

feature 7.10  Video Encoder Configuration
media   AddVideoEncoderConfiguration               O   7.10
media   GetCompatibleVideoEncoderConfigurations    O   7.10
media   GetGuaranteedNumberOfVideoEncoderInstances O   7.10
media   GetVideoEncoderConfiguration               O   7.10
media   GetVideoEncoderConfigurationOptions        M   7.10
media   GetVideoEncoderConfigurations              O   7.10
media   RemoveVideoEncoderConfiguration            O   7.10
media   SetVideoEncoderConfiguration               M   7.10

feature 7.11  Media Profile Configuration
media   CreateProfile                              M   7.11
media   DeleteProfile                              O   7.11
media   GetProfile                                 O   7.11

feature 7.12  Video Source Configuration
media   AddVideoSourceConfiguration                M   7.12
media   GetCompatibleVideoSourceConfigurations     M   7.12
media   GetVideoSourceConfiguration                O   7.12
media   GetVideoSourceConfigurationOptions         M   7.12
media   GetVideoSourceConfigurations               O   7.12
media   GetVideoSources                            O   7.12
media   RemoveVideoSourceConfiguration             O   7.12
media   SetVideoSourceConfiguration                M   7.12

feature 7.13  Metadata Configuration
media   AddMetadataConfiguration                   O   7.13
media   GetCompatibleMetadataConfigurations        O   7.13
media   GetMetadataConfiguration                   O   7.13
media   GetMetadataConfigurationOptions            M   7.13
media   GetMetadataConfigurations                  O   7.13
media   RemoveMetadataConfiguration                O   7.13
media   SetMetadataConfiguration                   M   7.13

feature 8.1   Video Streaming – MPEG4
media   SetSynchronizationPoint                    O   8.1

feature 8.3   PTZ
media   AddPTZConfiguration                        M   8.3
media   RemovePTZConfiguration                     O   8.3
ptz     ContinuousMove                             M   8.3
ptz     GetConfiguration                           O   8.3
ptz     GetConfigurationOptions                    O   8.3
ptz     GetConfigurations                          M   8.3
ptz     GetNode                                    M   8.3
ptz     GetNodes                                   M   8.3
ptz     GetStatus                                  O   8.3
ptz     SetConfiguration                           O   8.3
ptz     Stop                                       M   8.3

feature 8.4   PTZ – Absolute Positioning
ptz     AbsoluteMove                               M   8.4

feature 8.5   PTZ – Relative Positioning
ptz     RelativeMove                               M   8.5

feature 8.6   PTZ – Presets
ptz     GetPresets                                 M   8.6
ptz     GotoPreset                                 M   8.6
ptz     RemovePreset                               O   8.6
ptz     SetPreset                                  O   8.6

feature 8.7   PTZ – Home Position
ptz     GotoHomePosition                           M   8.7
ptz     SetHomePosition                            O   8.7

feature 8.8   PTZ – Auxiliary Command Position
ptz     SendAuxiliaryCommand                       M   8.8

feature 8.9   Audio Streaming
media   AddAudioEncoderConfiguration               M   8.9
media   AddAudioSourceConfiguration                M   8.9
media   GetAudioEncoderConfiguration               O   8.9
media   GetAudioEncoderConfigurationOptions        O   8.9
media   GetAudioEncoderConfigurations              O   8.9
media   GetAudioSourceConfiguration                O   8.9
media   GetAudioSourceConfigurationOptions         O   8.9
media   GetAudioSourceConfigurations               O   8.9
media   GetAudioSources                            O   8.9
media   GetCompatibleAudioEncoderConfigurations    M   8.9
media   GetCompatibleAudioSourceConfigurations     M   8.9
media   RemoveAudioEncoderConfiguration            O   8.9
media   RemoveAudioSourceConfiguration             O   8.9
media   SetAudioEncoderConfiguration               O   8.9
media   SetAudioSourceConfiguration                O   8.9

feature 8.12  Multicast Streaming
media   StartMulticastStreaming                    C   8.12
media   StopMulticastStreaming                     C   8.12

feature 8.13  Relay Outputs
device  GetRelayOutputs                            M   8.13
device  SetRelayOutputSettings                     M   8.13
device  SetRelayOutputState                        M   8.13

feature 8.14  NTP
device  GetNTP                                     M   8.14
device  SetNTP                                     M   8.14

feature 8.15  Dynamic DNS
device  GetDynamicDNS                              M   8.15
device  SetDynamicDNS                              M   8.15

feature 8.16  Zero Configuration
device  GetZeroConfiguration                       M   8.16
device  SetZeroConfiguration                       M   8.16

feature 8.17  IP Address Filtering
device  AddIPAddressFilter                         M   8.17
device  GetIPAddressFilter                         M   8.17
device  RemoveIPAddressFilter                      M   8.17
device  SetIPAddressFilter                         M   8.17

# 113 operations
