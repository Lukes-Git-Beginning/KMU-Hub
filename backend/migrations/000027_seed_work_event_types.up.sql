-- Seed notification event types for Work module (project management)
INSERT INTO event_types (module_id, event_key, display_name, description, default_priority, default_in_app, default_desktop_push, default_sound) VALUES
    ('work', 'work.task.created', 'New task assigned', 'A task has been assigned to you', 'normal', true, true, 'default'),
    ('work', 'work.task.assigned', 'Task reassigned', 'A task has been reassigned to you', 'normal', true, true, 'default'),
    ('work', 'work.task.status_changed', 'Task status changed', 'A task you follow changed status', 'low', true, false, ''),
    ('work', 'work.task.commented', 'New task comment', 'Someone commented on a task you follow', 'normal', true, true, 'default'),
    ('work', 'work.task.mentioned', 'Mentioned in task comment', 'Someone mentioned you in a task comment', 'normal', true, true, 'default');
