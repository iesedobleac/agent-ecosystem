import { Plugin } from "@opencode/plugin/tui"

// When a root session's context passes 200k tokens, and again every additional
// 100k, offers to compact it now, continue in a new session, or keep going.
const FIRST_WARNING = 200_000
const STEP = 100_000

type Tokens = { input?: number; cache?: { read?: number; write?: number } }
type Choice = "compact" | "new" | "keep"

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

    const failed = (title: string, error: unknown) =>
      context.ui.toast.show({
        title,
        message: error instanceof Error ? error.message : String(error),
        variant: "error",
        duration: 10_000,
      })

    const compact = async (sessionID: string) => {
      try {
        await context.client.session.compact({ sessionID })
        context.ui.toast.show({
          title: "Compacting",
          message: "The history is being summarized; the thread continues in this session.",
          variant: "info",
          duration: 5_000,
        })
      } catch (error) {
        failed("Compaction failed", error)
      }
    }

    // A new session in the same place, with the same agent and model, the
    // way OpenCode's own prompt creates one.
    const startNew = async (sessionID: string) => {
      const current = context.data.session.get(sessionID) as any
      try {
        const created = await context.client.session.create({
          location: current?.location,
          agent: current?.agent,
          model: current?.model,
        } as any)
        context.ui.router.navigate({ type: "session", sessionID: (created as any).id })
      } catch (error) {
        failed("Could not start a new session", error)
      }
    }

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
      const choice = await context.ui.dialog.select<Choice>({
        title: `This session uses ~${thousands}k tokens of context`,
        options: [
          { title: "Compact now", value: "compact", description: "Summarize the history and keep working in this session" },
          {
            title: "Start a new session",
            value: "new",
            description: "Fresh context, same project, agent and model; Engram keeps what was saved",
          },
          { title: "Keep going", value: "keep", description: `Ask again at ~${thousands + STEP / 1000}k` },
        ],
      })
      if (choice === "compact") await compact(sessionID)
      if (choice === "new") await startNew(sessionID)
    }

    const stop = context.data.on("session.execution.succeeded", (event: any) => {
      const sessionID = event?.data?.sessionID ?? event?.sessionID
      if (typeof sessionID !== "string") return
      void check(sessionID).catch(() => {})
    })
    return () => stop()
  },
})
