'use client';

import { FormEvent, useEffect, useState } from 'react';
import PageContainer from '@/components/layout/page-container';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Icons } from '@/components/icons';
import { posApi, unwrapList } from '@/lib/pos-api';

type Note = { id: string; title: string; content?: string; tags?: string[]; date_label?: string };

export default function NotebookPage() {
  const [notes, setNotes] = useState<Note[]>([]);
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  async function load() {
    try {
      const result = await posApi<unknown>('/notebook');
      setNotes(unwrapList<Note>(result.data));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load notes.');
    }
  }
  useEffect(() => {
    void load();
  }, []);
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi('/notebook', {
        method: 'POST',
        body: JSON.stringify({
          title: values.title,
          content: values.content,
          tags: String(values.tags || '')
            .split(',')
            .map((tag) => tag.trim())
            .filter(Boolean)
        })
      });
      setOpen(false);
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create note.');
    } finally {
      setPending(false);
    }
  }
  async function remove(id: string) {
    try {
      await posApi(`/notebook/${id}`, { method: 'DELETE' });
      setNotes((current) => current.filter((note) => note.id !== id));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not delete note.');
    }
  }
  const filtered = notes.filter((note) =>
    `${note.title} ${note.content || ''}`.toLowerCase().includes(query.toLowerCase())
  );
  return (
    <PageContainer
      pageTitle='Notebook'
      pageDescription='Keep shared notes and reminders for your store.'
      pageHeaderAction={
        <Button className='w-full sm:w-auto' onClick={() => setOpen(true)}>
          New note
        </Button>
      }
    >
      <div className='space-y-4'>
        {error && (
          <p role='alert' className='text-destructive text-sm'>
            {error}
          </p>
        )}
        <Card>
          <CardHeader>
            <CardTitle>Store notes</CardTitle>
            <CardDescription>Notes are shared with your active store team.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder='Search notes…'
            />
            <div className='grid gap-3 md:grid-cols-2'>
              {filtered.map((note) => (
                <Card key={note.id} size='sm'>
                  <CardHeader>
                    <div className='flex items-start justify-between gap-2'>
                      <div>
                        <CardTitle>{note.title}</CardTitle>
                        <CardDescription>{note.date_label || 'Note'}</CardDescription>
                      </div>
                      <Button
                        variant='ghost'
                        size='icon'
                        aria-label={`Delete ${note.title}`}
                        onClick={() => void remove(note.id)}
                      >
                        <Icons.trash className='text-destructive size-4' />
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <p className='text-muted-foreground whitespace-pre-wrap text-sm'>
                      {note.content || 'No content'}
                    </p>
                  </CardContent>
                </Card>
              ))}
              {!filtered.length && (
                <p className='text-muted-foreground py-8 text-center text-sm md:col-span-2'>
                  No notes found.
                </p>
              )}
            </div>
          </CardContent>
        </Card>
        {open && (
          <div
            className='fixed inset-0 z-50 flex items-end justify-center bg-black/60 p-0 sm:items-center sm:p-4'
            onMouseDown={(event) => {
              if (event.target === event.currentTarget) setOpen(false);
            }}
          >
            <Card className='min-h-[50svh] w-full rounded-b-none sm:min-h-0 sm:max-w-lg sm:rounded-xl'>
              <CardHeader className='flex flex-row items-center justify-between space-y-0'>
                <CardTitle>New note</CardTitle>
                <Button variant='ghost' size='icon' onClick={() => setOpen(false)}>
                  <Icons.close className='size-4' />
                </Button>
              </CardHeader>
              <CardContent>
                <form onSubmit={create} className='space-y-4'>
                  <Input name='title' placeholder='Note title' required />
                  <Textarea name='content' placeholder='Write a note…' rows={5} />
                  <Input name='tags' placeholder='Tags, separated by commas' />
                  <Button className='h-11 w-full' isLoading={pending}>
                    Save note
                  </Button>
                </form>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </PageContainer>
  );
}
