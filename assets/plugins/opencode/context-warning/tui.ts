import { Plugin } from "@opencode/plugin/tui"

// Shows a toast when a root session's context passes 200k tokens, and again
// every additional 100k, as a hint to continue in a new session.
const FIRST_WARNING = 200_000
const STEP = 100_000

type Tokens = { input?: number; cache?: { read?: number; write?: number } }

function contextSize(tokens: Tokens | undefined): number {
  if (!tokens) return 0
  return (tokens.input ?? 0) + (tokens.cache?.read ?? 0) + (tokens.cache?.write ?? 0)
}

function lastAssistantTokens(messages: readonly any[]): Tokens | undefined {
  let latest: any
  for (const message of messages) {
    if (message?.type !== "assistant" || !message.tokens) continue
    const created = message.time?.created ?? 0
    if (!latest || created > (latest.time?.created ?? 0)) latest = message
  }
  return latest?.tokens
}

export default Plugin.define({
  id: "local.context-warning",
  setup(context) {
    const warned = new Map<string, number>()

    const check = async (sessionID: string) => {
      const root = context.data.session.root(sessionID)
      if (root && root !== sessionID) return
      await context.data.session.message.sync(sessionID)
      const messages = context.data.session.message.list(sessionID) ?? []
      const size = contextSize(lastAssistantTokens(messages))
      if (size < FIRST_WARNING) return
      const level = Math.floor((size - FIRST_WARNING) / STEP)
      if ((warned.get(sessionID) ?? -1) >= level) return
      warned.set(sessionID, level)
      const thousands = Math.round(size / 1000)
      context.ui.toast.show({
        title: "Contexto grande",
        message: `Esta sesión ya usa ~${thousands}k tokens de contexto. Considera guardar el handoff y continuar en una sesión nueva.`,
        variant: "warning",
        duration: 15_000,
      })
    }

    const stop = context.data.on("session.execution.succeeded", (event: any) => {
      const sessionID = event?.data?.sessionID ?? event?.sessionID
      if (typeof sessionID !== "string") return
      void check(sessionID).catch(() => {})
    })
    return () => stop()
  },
})
