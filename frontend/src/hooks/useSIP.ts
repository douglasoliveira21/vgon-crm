import { useState, useRef, useCallback } from 'react'

export type SIPStatus = 'offline' | 'registering' | 'online' | 'error'
export type CallStatus = 'idle' | 'calling' | 'ringing' | 'in_call' | 'on_hold'

interface SIPConfig {
  server: string
  port: string
  domain: string
  user: string
  password: string
  displayName: string
  transport: string
  stunServer: string
}

interface UseSIPReturn {
  status: SIPStatus
  callStatus: CallStatus
  callDuration: number
  isMuted: boolean
  isOnHold: boolean
  remoteNumber: string
  register: (config: SIPConfig) => void
  unregister: () => void
  makeCall: (number: string) => void
  answerCall: () => void
  endCall: () => void
  toggleMute: () => void
  toggleHold: () => void
  sendDTMF: (digit: string) => void
  isIncoming: boolean
  incomingNumber: string
}

export function useSIP(): UseSIPReturn {
  const [status, setStatus] = useState<SIPStatus>('offline')
  const [callStatus, setCallStatus] = useState<CallStatus>('idle')
  const [callDuration, setCallDuration] = useState(0)
  const [isMuted, setIsMuted] = useState(false)
  const [isOnHold, setIsOnHold] = useState(false)
  const [remoteNumber, setRemoteNumber] = useState('')
  const [isIncoming, setIsIncoming] = useState(false)
  const [incomingNumber, setIncomingNumber] = useState('')

  const uaRef = useRef<any>(null)
  const sessionRef = useRef<any>(null)
  const timerRef = useRef<NodeJS.Timeout | null>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  const startTimer = () => {
    setCallDuration(0)
    timerRef.current = setInterval(() => setCallDuration(d => d + 1), 1000)
  }

  const stopTimer = () => {
    if (timerRef.current) clearInterval(timerRef.current)
    timerRef.current = null
  }

  const setupRemoteAudio = (session: any) => {
    // Create audio element for remote audio
    if (!audioRef.current) {
      audioRef.current = new Audio()
      audioRef.current.autoplay = true
      document.body.appendChild(audioRef.current)
    }

    // For SIP.js 0.21.x - get remote media stream
    if (session.sessionDescriptionHandler) {
      const pc = session.sessionDescriptionHandler.peerConnection
      if (pc) {
        pc.ontrack = (event: RTCTrackEvent) => {
          if (audioRef.current && event.streams[0]) {
            audioRef.current.srcObject = event.streams[0]
          }
        }
      }
    }
  }

  const register = useCallback(async (config: SIPConfig) => {
    try {
      // Dynamically import sip.js (only in browser)
      const SIP = await import('sip.js')

      setStatus('registering')

      const wsServer = `${config.transport === 'WSS' ? 'wss' : 'ws'}://${config.server}:${config.port}/ws`
      const uri = SIP.UserAgent.makeURI(`sip:${config.user}@${config.domain}`)

      if (!uri) {
        setStatus('error')
        return
      }

      const transportOptions = {
        server: wsServer,
        traceSip: false,
      }

      const ua = new SIP.UserAgent({
        uri,
        transportOptions,
        authorizationUsername: config.user,
        authorizationPassword: config.password,
        displayName: config.displayName,
        sessionDescriptionHandlerFactoryOptions: {
          peerConnectionConfiguration: {
            iceServers: [{ urls: config.stunServer }],
          },
        },
      })

      const registerer = new SIP.Registerer(ua)

      // Handle incoming calls
      ua.delegate = {
        onInvite: (invitation: any) => {
          sessionRef.current = invitation
          const from = invitation.remoteIdentity?.uri?.user || 'Desconhecido'
          setIncomingNumber(from)
          setIsIncoming(true)

          invitation.stateChange.addListener((state: any) => {
            if (state === SIP.SessionState.Terminated) {
              setCallStatus('idle')
              setIsIncoming(false)
              stopTimer()
            }
          })
        },
      }

      await ua.start()

      registerer.stateChange.addListener((state: any) => {
        switch (state) {
          case SIP.RegistererState.Registered:
            setStatus('online')
            break
          case SIP.RegistererState.Unregistered:
            setStatus('offline')
            break
          default:
            break
        }
      })

      await registerer.register()
      uaRef.current = { ua, registerer }

    } catch (error) {
      console.error('SIP Registration failed:', error)
      setStatus('error')
    }
  }, [])

  const unregister = useCallback(async () => {
    try {
      if (uaRef.current?.registerer) {
        await uaRef.current.registerer.unregister()
      }
      if (uaRef.current?.ua) {
        await uaRef.current.ua.stop()
      }
      uaRef.current = null
      setStatus('offline')
    } catch {
      setStatus('offline')
    }
  }, [])

  const makeCall = useCallback(async (number: string) => {
    if (!uaRef.current?.ua || status !== 'online') return

    try {
      const SIP = await import('sip.js')
      const target = SIP.UserAgent.makeURI(`sip:${number}@${uaRef.current.ua.configuration.uri.host}`)
      if (!target) return

      const inviter = new SIP.Inviter(uaRef.current.ua, target, {
        sessionDescriptionHandlerOptions: {
          constraints: { audio: true, video: false },
        },
      })

      sessionRef.current = inviter
      setRemoteNumber(number)
      setCallStatus('calling')

      inviter.stateChange.addListener((state: any) => {
        switch (state) {
          case SIP.SessionState.Establishing:
            setCallStatus('ringing')
            break
          case SIP.SessionState.Established:
            setCallStatus('in_call')
            startTimer()
            setupRemoteAudio(inviter)
            break
          case SIP.SessionState.Terminated:
            setCallStatus('idle')
            setRemoteNumber('')
            stopTimer()
            break
        }
      })

      await inviter.invite()
    } catch (error) {
      console.error('Call failed:', error)
      setCallStatus('idle')
    }
  }, [status])

  const answerCall = useCallback(async () => {
    if (!sessionRef.current) return
    try {
      await sessionRef.current.accept({
        sessionDescriptionHandlerOptions: {
          constraints: { audio: true, video: false },
        },
      })
      setIsIncoming(false)
      setCallStatus('in_call')
      setRemoteNumber(incomingNumber)
      startTimer()
      setupRemoteAudio(sessionRef.current)
    } catch (error) {
      console.error('Answer failed:', error)
    }
  }, [incomingNumber])

  const endCall = useCallback(async () => {
    if (!sessionRef.current) return
    try {
      // Depending on session state, use bye or cancel
      if (sessionRef.current.state === 'Established') {
        sessionRef.current.bye()
      } else {
        sessionRef.current.cancel()
      }
    } catch {}
    sessionRef.current = null
    setCallStatus('idle')
    setRemoteNumber('')
    setIsIncoming(false)
    stopTimer()
  }, [])

  const toggleMute = useCallback(() => {
    if (!sessionRef.current?.sessionDescriptionHandler) return
    const pc = sessionRef.current.sessionDescriptionHandler.peerConnection
    if (pc) {
      pc.getSenders().forEach((sender: RTCRtpSender) => {
        if (sender.track?.kind === 'audio') {
          sender.track.enabled = isMuted
        }
      })
      setIsMuted(!isMuted)
    }
  }, [isMuted])

  const toggleHold = useCallback(async () => {
    if (!sessionRef.current) return
    try {
      if (isOnHold) {
        // Unhold - reinvite
        setIsOnHold(false)
      } else {
        // Hold
        setIsOnHold(true)
      }
    } catch {}
  }, [isOnHold])

  const sendDTMF = useCallback((digit: string) => {
    if (!sessionRef.current) return
    try {
      sessionRef.current.info({
        requestOptions: {
          body: { contentDisposition: 'render', contentType: 'application/dtmf-relay', content: `Signal=${digit}\nDuration=160` },
        },
      })
    } catch {}
  }, [])

  return {
    status,
    callStatus,
    callDuration,
    isMuted,
    isOnHold,
    remoteNumber,
    register,
    unregister,
    makeCall,
    answerCall,
    endCall,
    toggleMute,
    toggleHold,
    sendDTMF,
    isIncoming,
    incomingNumber,
  }
}
