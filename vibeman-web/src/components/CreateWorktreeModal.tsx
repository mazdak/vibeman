import React, { useEffect } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Checkbox } from './ui/checkbox';
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from './ui/form';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Loader2, GitBranch, Plus, AlertCircle } from 'lucide-react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { postWorktreesMutation } from '@/generated/api/@tanstack/react-query.gen';
import { useToast } from './ui/toast';

const worktreeSchema = z.object({
  name: z.string()
    .min(1, 'Worktree name is required')
    .max(100, 'Name too long')
    .regex(/^[a-zA-Z0-9-_]+$/, 'Name can only contain letters, numbers, hyphens, and underscores'),
  repository_id: z.string().min(1, 'Repository is required'),
  auto_start: z.boolean().default(true),
});

type WorktreeFormData = z.infer<typeof worktreeSchema>;

interface CreateWorktreeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  repositories: Array<{ id: string; name: string; }>;
  selectedRepositoryId?: string;
}

export function CreateWorktreeModal({ 
  open, 
  onOpenChange, 
  onSuccess, 
  repositories,
  selectedRepositoryId 
}: CreateWorktreeModalProps) {
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const createWorktreeMutation = useMutation({
    ...postWorktreesMutation(),
    onSuccess: () => {
      // Invalidate worktree queries to refetch data
      queryClient.invalidateQueries({
        queryKey: ['getWorktrees'],
      });
      onSuccess();
    },
  });

  const form = useForm<WorktreeFormData>({
    resolver: zodResolver(worktreeSchema),
    defaultValues: {
      name: '',
      repository_id: selectedRepositoryId || '',
      auto_start: true,
    },
  });

  useEffect(() => {
    if (selectedRepositoryId) {
      form.setValue('repository_id', selectedRepositoryId);
    }
  }, [selectedRepositoryId, form]);

  const onSubmit = (data: WorktreeFormData) => {
    createWorktreeMutation.mutate(
      {
        body: {
          repository_id: data.repository_id,
          name: data.name,
          auto_start: data.auto_start,
        },
      },
      {
        onSuccess: () => {
          // Show success toast
          toast({
            title: 'Worktree created successfully',
            description: `Worktree "${data.name}" has been created${data.auto_start ? ' and is starting up' : ''}.`,
            duration: 5000,
          });
          
          form.reset();
          onOpenChange(false);
        },
        onError: (error) => {
          console.error('Failed to create worktree:', error);
          
          // Provide more specific error messages based on error type
          let errorMessage = 'Failed to create worktree. Please try again.';
          
          if (error?.message) {
            // Check for specific error patterns
            if (error.message.includes('already exists')) {
              errorMessage = 'A worktree with this name already exists. Please choose a different name.';
            } else if (error.message.includes('permission')) {
              errorMessage = 'Permission denied. Please check your repository permissions.';
            } else if (error.message.includes('network') || error.message.includes('fetch')) {
              errorMessage = 'Network error. Please check your connection and try again.';
            } else {
              errorMessage = error.message;
            }
          }
          
          form.setError('root', {
            message: errorMessage,
          });
        }
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[550px]">
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold flex items-center gap-2">
            <GitBranch className="w-5 h-5" />
            Create New Worktree
          </DialogTitle>
          <DialogDescription>
            Create an isolated development environment with its own container.
          </DialogDescription>
        </DialogHeader>
        
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="repository_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Repository</FormLabel>
                  <Select 
                    value={field.value} 
                    onValueChange={field.onChange}
                    disabled={!!selectedRepositoryId}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select a repository" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {repositories.map((repo) => (
                        <SelectItem key={repo.id} value={repo.id}>
                          {repo.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Worktree Name</FormLabel>
                  <FormControl>
                    <Input 
                      placeholder="feature-authentication" 
                      {...field} 
                      pattern="[a-zA-Z0-9-_]+"
                      title="Only letters, numbers, hyphens, and underscores allowed"
                    />
                  </FormControl>
                  <FormDescription>
                    A descriptive name for your worktree (letters, numbers, hyphens, underscores only)
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="auto_start"
              render={({ field }) => (
                <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-lg border p-4 bg-muted/50">
                  <FormControl>
                    <Checkbox
                      checked={field.value}
                      onCheckedChange={field.onChange}
                      className="mt-0.5"
                    />
                  </FormControl>
                  <div className="space-y-1 leading-none">
                    <FormLabel>Auto-start container</FormLabel>
                    <FormDescription>
                      Automatically start the development container after creating the worktree
                    </FormDescription>
                  </div>
                </FormItem>
              )}
            />
            
            {form.formState.errors.root && (
              <div className="text-sm text-destructive bg-destructive/10 border border-destructive/50 rounded-lg px-3 py-2 flex items-start gap-2">
                <AlertCircle className="w-4 h-4 flex-shrink-0 mt-0.5" />
                <span>{form.formState.errors.root.message}</span>
              </div>
            )}
            
            <DialogFooter className="flex gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={createWorktreeMutation.isPending}
              >
                Cancel
              </Button>
              <Button 
                type="submit" 
                disabled={createWorktreeMutation.isPending}
                className="bg-gradient-to-r from-cyan-500 to-purple-500 text-white hover:from-cyan-600 hover:to-purple-600 disabled:opacity-50"
              >
                {createWorktreeMutation.isPending ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Creating...
                  </>
                ) : (
                  <>
                    <Plus className="w-4 h-4 mr-2" />
                    Create Worktree
                  </>
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}