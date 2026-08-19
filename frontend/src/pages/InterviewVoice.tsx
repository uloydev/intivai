import { useEffect, useRef, useState } from "react"
import { useParams, Link } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  MicrophoneStage,
  MicrophoneSlash,
  PhoneDisconnect,
  Sparkle,
  SpeakerHigh,
  ArrowLeft,
  Circle,
} from "@phosphor-icons/react"
import { Code2 } from "lucide-react"
import { CodingSandbox } from "@/components/sandbox/CodingSandbox"
import { aiReview, runCode } from "@/lib/sandbox"
import { cn } from "@/lib/utils"
import { toast } from "sonner"
import type { SandboxLanguage, SandboxTestCase } from "@/types/api"

export function InterviewVoicePage() {
  const { id } = useParams<{ id: string }>()
  const [isStarted, setIsStarted] = useState(false)
  const [status, setStatus] = useState("Ready to connect")
  const [isMuted, setIsMuted] = useState(false)
  const [liveCaption, setLiveCaption] = useState<string>("Click 'Start Interview' to initiate the WebRTC voice stream.")
  const [callDuration, setCallDuration] = useState(0)
  const [showSandbox, setShowSandbox] = useState(false)

  const pcRef = useRef<RTCPeerConnection | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const isStartedRef = useRef(false)

  useEffect(() => {
    isStartedRef.current = isStarted
  }, [isStarted])

  // Unmount cleanup: navigating away must stop the mic, close the peer
  // connection and the socket (the old code leaked all three).
  useEffect(() => {
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
      if (streamRef.current) streamRef.current.getTracks().forEach((track) => track.stop())
      if (pcRef.current) pcRef.current.close()
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) wsRef.current.close()
    }
  }, [])

  const startVoiceInterview = async () => {
    try {
      setStatus("Requesting microphone permission...")
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
      })
      streamRef.current = stream

      setStatus("Initializing WebRTC peer connection...")
      const pc = new RTCPeerConnection({
        iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
      })
      pcRef.current = pc

      stream.getTracks().forEach((track) => pc.addTrack(track, stream))

      pc.ontrack = (event) => {
        if (audioRef.current && event.streams[0]) {
          audioRef.current.srcObject = event.streams[0]
          // autoplay policy — the user has already clicked "Start Call"
          audioRef.current.play().catch(() => {})
        }
      }

      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
      const token = localStorage.getItem("intivai_token") ?? ""
      const wsUrl = `${protocol}//${window.location.host}/api/v1/interviews/${id}/voice?ticket=${encodeURIComponent(token)}`
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = async () => {
        setStatus("Creating WebRTC audio offer...")
        const offer = await pc.createOffer()
        await pc.setLocalDescription(offer)
        ws.send(JSON.stringify({ type: "offer", sdp: offer.sdp }))
      }

      ws.onmessage = async (event) => {
        try {
          const msg = JSON.parse(event.data)
          if (msg.type === "answer") {
            await pc.setRemoteDescription(new RTCSessionDescription({ type: "answer", sdp: msg.sdp }))
            setStatus("Connected — AI Interview in progress")
            setIsStarted(true)
          } else if (msg.type === "caption") {
            setLiveCaption(msg.text)
          } else if (msg.type === "candidate") {
            await pc.addIceCandidate(new RTCIceCandidate(msg.candidate))
          } else if (msg.type === "audio" && msg.data) {
            // MVP demo path: the sidecar-less pipeline ships TTS audio as
            // base64 frames over the WS (Opus-over-RTP is deferred Phase 5).
            const blob = new Blob(
              [Uint8Array.from(atob(msg.data), (ch) => ch.charCodeAt(0))],
              { type: "audio/mpeg" }
            )
            if (audioRef.current) {
              if (audioRef.current.src) URL.revokeObjectURL(audioRef.current.src)
              audioRef.current.src = URL.createObjectURL(blob)
              void audioRef.current.play().catch(() => {
                // autoplay policy — user has already clicked "Start Call"
              })
            }
          }
        } catch (err) {
          // Malformed signaling frame — drop it and keep the stream alive.
          console.error("Failed to parse voice signaling frame", err)
        }
      }

      pc.onicecandidate = (event) => {
        if (event.candidate && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "candidate", data: JSON.stringify(event.candidate) }))
        }
      }

      ws.onerror = () => {
        setStatus("Voice service error (simulation fallback active)")
        setIsStarted(true)
      }

      ws.onclose = () => {
        // Read the ref, not the render-closure value (always false here).
        if (isStartedRef.current) {
          setStatus("Call completed")
        }
      }
    } catch {
      setStatus("Microphone permission denied or device not found")
      toast.error("Microphone access required for voice interview.")
    }
  }

  const toggleMute = () => {
    if (streamRef.current) {
      const audioTrack = streamRef.current.getAudioTracks()[0]
      if (audioTrack) {
        audioTrack.enabled = isMuted
        setIsMuted(!isMuted)
        toast.info(isMuted ? "Microphone unmuted" : "Microphone muted")
      }
    }
  }

  const endCall = () => {
    if (timerRef.current) clearInterval(timerRef.current)
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop())
    }
    if (pcRef.current) {
      pcRef.current.close()
    }
    if (wsRef.current) {
      wsRef.current.close()
    }
    setIsStarted(false)
    setStatus("Call ended")
    toast.success("Interview session completed.")
  }

  useEffect(() => {
    if (isStarted) {
      timerRef.current = setInterval(() => {
        setCallDuration((prev) => prev + 1)
      }, 1000)
    } else {
      if (timerRef.current) clearInterval(timerRef.current)
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [isStarted])

  const formatDuration = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`
  }

  const handleExecuteSandbox = (language: SandboxLanguage, code: string, testCases: SandboxTestCase[]) =>
    runCode(language, code, testCases)

  const handleAIReview = (language: SandboxLanguage, code: string) =>
    aiReview(language, code, liveCaption || "Voice Interview Coding Problem")

  return (
    <div className="flex flex-col h-screen bg-background text-foreground overflow-hidden">
      <audio ref={audioRef} autoPlay className="hidden" />

      {/* Top Header */}
      <header className="flex h-14 items-center justify-between border-b border-border/80 bg-card/60 backdrop-blur-md px-5 shrink-0">
        <div className="flex items-center gap-3">
          <Link
            to="/interviews"
            className="flex h-8 w-8 items-center justify-center rounded-lg border border-border/70 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="flex items-center gap-2">
            <span className="font-display font-bold text-sm tracking-tight text-foreground">
              Intivai Voice & Coding Room
            </span>
            {isStarted && (
              <Badge variant="outline" className="gap-1 border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 text-xs py-0">
                <Circle className="h-2 w-2 fill-current animate-ping" />
                {formatDuration(callDuration)}
              </Badge>
            )}
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Button
            variant={showSandbox ? "secondary" : "outline"}
            size="sm"
            onClick={() => setShowSandbox(!showSandbox)}
            className="text-xs h-8 gap-1.5 border-border/70 shadow-sm"
          >
            <Code2 className="w-3.5 h-3.5 text-indigo-400" />
            <span>{showSandbox ? "Hide Code Sandbox" : "Code Sandbox"}</span>
          </Button>
        </div>
      </header>

      {/* Main Workspace Body */}
      <div className="flex-1 flex min-h-0 overflow-hidden">
        {/* Voice Orb Area */}
        <div
          className={cn(
            "flex flex-col items-center justify-center p-6 transition-all duration-300 overflow-y-auto",
            showSandbox ? "w-[45%] border-r border-border" : "w-full"
          )}
        >
          <Card className="w-full max-w-md glass border-primary/20 shadow-2xl shadow-primary/10 overflow-hidden relative">
            <CardHeader className="text-center pb-2">
              <div className="mx-auto mb-2 flex items-center justify-center gap-1.5">
                <Badge variant="outline" className="gap-1 border-primary/30 bg-primary/5 text-primary text-xs py-0.5">
                  <Sparkle className="h-3 w-3" weight="fill" /> AI Voice Session
                </Badge>
                <Badge variant="secondary" className="text-xs">
                  WebRTC Full Duplex
                </Badge>
              </div>
              <CardTitle className="font-display text-xl font-bold tracking-tight">Intivai Voice Evaluator</CardTitle>
              <CardDescription className="text-xs">
                Whisper STT + Edge Neural Voice Synthesis
              </CardDescription>
            </CardHeader>

            <CardContent className="flex flex-col items-center justify-center gap-5 p-5">
              {/* Animated Center Orb */}
              <div className="relative my-2 flex items-center justify-center">
                {isStarted && (
                  <>
                    <div className="absolute h-40 w-40 rounded-full bg-primary/10 animate-ping duration-1000" />
                    <div className="absolute h-32 w-32 rounded-full bg-primary/20 animate-pulse" />
                  </>
                )}
                <div
                  className={cn(
                    "relative flex h-24 w-24 items-center justify-center rounded-full shadow-xl transition-all duration-500",
                    isStarted
                      ? "bg-primary text-primary-foreground shadow-primary/30 scale-105"
                      : "bg-muted text-muted-foreground border border-border/80"
                  )}
                >
                  {isMuted ? (
                    <MicrophoneSlash size={40} weight="fill" className="text-destructive" />
                  ) : (
                    <MicrophoneStage size={40} weight={isStarted ? "fill" : "regular"} />
                  )}
                </div>
              </div>

              {/* Status Pill */}
              <div className="flex items-center gap-2 rounded-full border border-border/60 bg-muted/40 px-3.5 py-1 text-xs">
                <span
                  className={cn(
                    "h-2 w-2 rounded-full",
                    isStarted ? (isMuted ? "bg-amber-400" : "bg-emerald-400 animate-pulse") : "bg-muted-foreground"
                  )}
                />
                <span className="font-medium text-foreground">{status}</span>
              </div>

              {/* Live Audio Caption Display */}
              <div className="w-full rounded-xl border border-border/50 bg-background/60 p-3 text-center">
                <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-0.5 flex items-center justify-center gap-1.5">
                  <SpeakerHigh className="h-3.5 w-3.5 text-primary" /> Live Voice Caption
                </p>
                <p className="text-xs font-medium text-foreground leading-relaxed italic">
                  "{liveCaption}"
                </p>
              </div>

              {/* Voice Controls */}
              <div className="flex items-center gap-3 mt-1">
                {!isStarted ? (
                  <Button
                    variant="gradient"
                    size="lg"
                    className="gap-2 px-6 shadow-lg shadow-primary/25 rounded-full"
                    onClick={startVoiceInterview}
                  >
                    <MicrophoneStage size={20} weight="fill" />
                    <span>Start Voice Interview</span>
                  </Button>
                ) : (
                  <>
                    <Button
                      variant={isMuted ? "destructive" : "secondary"}
                      size="icon"
                      className="h-12 w-12 rounded-full shadow-md"
                      onClick={toggleMute}
                      title={isMuted ? "Unmute mic" : "Mute mic"}
                    >
                      {isMuted ? <MicrophoneSlash size={22} weight="fill" /> : <MicrophoneStage size={22} />}
                    </Button>
                    <Button
                      variant="destructive"
                      size="icon"
                      className="h-12 w-12 rounded-full shadow-md shadow-destructive/25"
                      onClick={endCall}
                      title="End Voice Call"
                    >
                      <PhoneDisconnect size={22} weight="fill" />
                    </Button>
                  </>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right Split Column: Live Coding Sandbox */}
        {showSandbox && (
          <div className="w-[55%] h-full bg-neutral-950 flex flex-col min-w-0">
            <CodingSandbox
              onExecute={handleExecuteSandbox}
              onRequestAIReview={handleAIReview}
            />
          </div>
        )}
      </div>
    </div>
  )
}
