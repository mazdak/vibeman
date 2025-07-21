import React, { useEffect } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from './ui/dialog';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Textarea } from './ui/textarea';
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from './ui/form';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Loader2 } from 'lucide-react';

const repositorySchema = z.object({
  name: z.string().min(1, 'Repository name is required').max(100, 'Name too long'),
  repository_url: z.string().min(1, 'Repository path or URL is required').refine(
    (path) => {
      // Allow local paths (absolute or relative)
      if (path.startsWith('/') || path.startsWith('./') || path.startsWith('../') || /^[a-zA-Z]:/.test(path)) {
        return true;
      }
      // Allow git URLs
      if (path.startsWith('https://') || path.startsWith('http://') || path.startsWith('git@') || path.startsWith('ssh://') || path.includes(':')) {
        return true;
      }
      // Allow simple relative paths without ./ prefix
      if (!path.includes('://')) {
        return true;
      }
      return false;
    },
    'Must be a valid repository path or Git URL'
  ),
  description: z.string().optional(),
});

type RepositoryFormData = z.infer<typeof repositorySchema>;

interface CreateRepositoryModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
  createProject: (data: any) => void;
  isCreating: boolean;
}

export function CreateRepositoryModal({ open, onOpenChange, onSuccess, createProject, isCreating }: CreateRepositoryModalProps) {

  const form = useForm<RepositoryFormData>({
    resolver: zodResolver(repositorySchema),
    defaultValues: {
      name: '',
      repository_url: '',
      description: '',
    },
  });

  const onSubmit = (data: RepositoryFormData) => {
    createProject({
      body: {
        name: data.name,
        git_url: data.repository_url,
        repository_url: data.repository_url,
        description: data.description,
      }
    });
  };

  // Call onSuccess when creation is complete
  useEffect(() => {
    if (open && !isCreating && form.formState.isSubmitSuccessful) {
      form.reset();
      onSuccess();
    }
  }, [isCreating, open, form, onSuccess]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold">Add New Repository</DialogTitle>
          <DialogDescription>
            Add a new Git repository to manage with Vibeman.
          </DialogDescription>
        </DialogHeader>
        
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Repository Name</FormLabel>
                  <FormControl>
                    <Input 
                      placeholder="my-awesome-project" 
                      {...field} 
                    />
                  </FormControl>
                  <FormDescription>
                    A friendly name for your repository
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            
            <FormField
              control={form.control}
              name="repository_url"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Repository URL or Path</FormLabel>
                  <FormControl>
                    <Input 
                      placeholder="https://github.com/username/repo.git or /path/to/repo" 
                      {...field} 
                    />
                  </FormControl>
                  <FormDescription>
                    The Git clone URL or local path to your repository
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description (Optional)</FormLabel>
                  <FormControl>
                    <Textarea 
                      placeholder="A brief description of the repository" 
                      className="resize-none"
                      {...field} 
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            
            {form.formState.errors.root && (
              <div className="text-sm text-destructive bg-destructive/10 border border-destructive/50 rounded-lg px-3 py-2">
                {form.formState.errors.root.message}
              </div>
            )}
            
            <DialogFooter className="flex gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={isCreating}
                className=""
              >
                Cancel
              </Button>
              <Button 
                type="submit" 
                disabled={isCreating}
                className="bg-gradient-to-r from-cyan-500 to-purple-500 text-white hover:from-cyan-600 hover:to-purple-600 disabled:opacity-50"
              >
                {isCreating ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Creating...
                  </>
                ) : (
                  'Create Repository'
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}