'use client';

import { FormEvent, useEffect, useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Icons } from '@/components/icons';
import { posApi } from '@/lib/pos-api';

interface ConversationSummary {
  id: string;
  title?: string;
  first_message_preview?: string;
  updated_at?: string;
  message_count?: number;
}
interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  created_at?: string;
  sources_used?: string[];
}
interface ConversationDetail extends ConversationSummary {
  messages?: ChatMessage[];
}
interface ListResponse {
  conversations?: ConversationSummary[];
  items?: ConversationSummary[];
}
interface ChatResponse {
  conversation?: ConversationDetail;
  conversation_id?: string;
  id?: string;
  title?: string;
  message?: ChatMessage;
}

const suggestions = [
  'How are sales performing today?',
  'Which products need restocking?',
  'Who owes us money?',
  'Summarise this month’s expenses'
];

export function ZiadaAiChat() {
  const [conversations, setConversations] = useState<ConversationSummary[]>([]);
  const [active, setActive] = useState<ConversationDetail | null>(null);
  const [draft, setDraft] = useState('');
  const [search, setSearch] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function loadConversations() {
    const result = await posApi<ListResponse>('/ai/conversations');
    setConversations(result.data.conversations || result.data.items || []);
  }
  useEffect(() => {
    void loadConversations().catch((reason) =>
      setError(reason instanceof Error ? reason.message : 'Could not load conversations.')
    );
  }, []);
  async function openConversation(id: string) {
    setError('');
    try {
      const result = await posApi<ConversationDetail>(`/ai/conversations/${id}`);
      setActive(result.data);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not open conversation.');
    }
  }
  async function send(event?: FormEvent) {
    event?.preventDefault();
    const message = draft.trim();
    if (!message || busy) return;
    setDraft('');
    setBusy(true);
    setError('');
    try {
      const result = active
        ? await posApi<ChatResponse>(`/ai/conversations/${active.id}/chat`, {
            method: 'POST',
            body: JSON.stringify({ message })
          })
        : await posApi<ChatResponse>('/ai/chat', {
            method: 'POST',
            body: JSON.stringify({ message, title: message.slice(0, 60) })
          });
      const reply = result.data.message;
      const userMessage: ChatMessage = { id: crypto.randomUUID(), role: 'user', content: message };
      const next = active
        ? {
            ...active,
            messages: [...(active.messages || []), userMessage, ...(reply ? [reply] : [])]
          }
        : result.data.conversation || {
            id: result.data.conversation_id || result.data.id || crypto.randomUUID(),
            title: result.data.title || message.slice(0, 60),
            first_message_preview: message,
            messages: [userMessage, ...(reply ? [reply] : [])]
          };
      if (next) setActive(next);
      await loadConversations();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Ziada AI could not reply.');
      setDraft(message);
    } finally {
      setBusy(false);
    }
  }
  const filtered = useMemo(
    () =>
      conversations.filter((item) =>
        `${item.title || ''} ${item.first_message_preview || ''}`
          .toLowerCase()
          .includes(search.toLowerCase())
      ),
    [conversations, search]
  );
  return (
    <div className='flex min-h-[calc(100svh-8rem)] min-w-0 flex-1 overflow-hidden rounded-2xl border bg-background shadow-sm'>
      <aside className='hidden w-72 shrink-0 border-r bg-muted/10 md:flex md:flex-col'>
        <div className='flex items-center justify-between border-b p-4'>
          <div>
            <p className='font-semibold'>Ziada AI</p>
            <p className='text-muted-foreground text-xs'>Business assistant</p>
          </div>
          <Button
            size='icon'
            variant='outline'
            aria-label='New conversation'
            onClick={() => setActive(null)}
          >
            <Icons.add className='h-4 w-4' />
          </Button>
        </div>
        <div className='p-3'>
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder='Search conversations…'
          />
        </div>
        <div className='min-h-0 flex-1 space-y-1 overflow-y-auto px-2 pb-3'>
          {filtered.map((item) => (
            <button
              key={item.id}
              className={`hover:bg-muted/60 w-full rounded-lg p-3 text-left ${active?.id === item.id ? 'bg-muted' : ''}`}
              onClick={() => void openConversation(item.id)}
            >
              <p className='truncate text-sm font-medium'>{item.title || 'Conversation'}</p>
              <p className='text-muted-foreground mt-1 truncate text-xs'>
                {item.first_message_preview || 'No messages yet'}
              </p>
            </button>
          ))}
          {!filtered.length && (
            <p className='text-muted-foreground px-2 py-6 text-center text-xs'>
              No conversations yet.
            </p>
          )}
        </div>
      </aside>
      <main className='flex min-w-0 flex-1 flex-col'>
        <header className='flex items-center justify-between border-b px-4 py-3 sm:px-6'>
          <div className='flex items-center gap-3'>
            <div className='bg-muted grid h-9 w-9 place-items-center overflow-hidden rounded-xl'>
              <img src='/ziada-ai-avatar.png' alt='' className='h-full w-full object-cover' />
            </div>
            <div>
              <h1 className='font-semibold'>{active?.title || 'Ziada AI'}</h1>
              <p className='text-muted-foreground text-xs'>
                Ask about your shop, sales, stock, and customers
              </p>
            </div>
          </div>
          <Button className='md:hidden' size='sm' variant='outline' onClick={() => setActive(null)}>
            New chat
          </Button>
        </header>
        <div className='min-h-0 flex-1 overflow-y-auto p-4 sm:p-8'>
          <div className='mx-auto flex max-w-3xl flex-col gap-5'>
            {!active?.messages?.length ? (
              <div className='flex min-h-[45vh] flex-col items-center justify-center text-center'>
                <div className='bg-primary/10 text-primary mb-4 grid h-14 w-14 place-items-center rounded-2xl'>
                  <Icons.sparkles className='h-7 w-7' />
                </div>
                <h2 className='text-2xl font-semibold tracking-tight'>How can I help your shop?</h2>
                <p className='text-muted-foreground mt-2 max-w-md text-sm'>
                  Ask Ziada AI to explain performance, find opportunities, or guide you through a
                  task.
                </p>
                <div className='mt-6 grid w-full gap-2 sm:grid-cols-2'>
                  {suggestions.map((suggestion) => (
                    <button
                      key={suggestion}
                      className='hover:border-primary/50 rounded-xl border p-3 text-left text-sm transition-colors'
                      onClick={() => {
                        setDraft(suggestion);
                      }}
                    >
                      {suggestion}
                      <Icons.arrowRight className='text-muted-foreground float-right mt-0.5 h-4 w-4' />
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              active.messages.map((message) => (
                <div
                  key={message.id}
                  className={`flex gap-3 ${message.role === 'user' ? 'justify-end' : ''}`}
                >
                  <div
                    className={`max-w-[88%] rounded-2xl px-4 py-3 text-sm leading-6 ${message.role === 'user' ? 'bg-primary text-primary-foreground rounded-br-md' : 'bg-muted rounded-bl-md'}`}
                  >
                    <p className='whitespace-pre-wrap'>{message.content}</p>
                    {message.sources_used?.length ? (
                      <p className='text-muted-foreground mt-2 text-xs'>
                        Sources: {message.sources_used.join(', ')}
                      </p>
                    ) : null}
                  </div>
                </div>
              ))
            )}
            {busy && (
              <div className='bg-muted text-muted-foreground w-fit rounded-2xl rounded-bl-md px-4 py-3 text-sm'>
                Ziada is thinking…
              </div>
            )}
            {error && (
              <p role='alert' className='text-destructive text-sm'>
                {error}
              </p>
            )}
          </div>
        </div>
        <form onSubmit={send} className='border-t p-3 sm:p-4'>
          <div className='mx-auto flex max-w-3xl items-end gap-2 rounded-2xl border bg-muted/20 p-2'>
            <Textarea
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault();
                  void send();
                }
              }}
              rows={1}
              placeholder='Ask Ziada AI anything about your business…'
              className='min-h-10 resize-none border-0 bg-transparent shadow-none focus-visible:ring-0'
            />
            <Button
              type='submit'
              size='icon'
              disabled={!draft.trim() || busy}
              aria-label='Send message'
            >
              <Icons.send className='h-4 w-4' />
            </Button>
          </div>
          <p className='text-muted-foreground mx-auto mt-2 max-w-3xl text-center text-[11px]'>
            Enter to send · Shift + Enter for a new line
          </p>
        </form>
      </main>
    </div>
  );
}
