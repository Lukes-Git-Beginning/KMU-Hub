DELETE FROM permissions
WHERE resource IN ('produktion:order', 'produktion:booking', 'produktion:plan');
