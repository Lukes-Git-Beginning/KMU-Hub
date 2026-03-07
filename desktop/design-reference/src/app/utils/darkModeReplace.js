// This is a utility script that would help replace dark mode classes
// Not executed in the app, just for documentation

const replacements = {
  'bg-white border border-slate-200': 'bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700',
  'text-slate-900': 'text-slate-900 dark:text-white',
  'text-slate-600': 'text-slate-600 dark:text-slate-300',
  'text-slate-500': 'text-slate-500 dark:text-slate-400',
  'text-slate-700': 'text-slate-700 dark:text-slate-200',
  'hover:bg-slate-50': 'hover:bg-slate-50 dark:hover:bg-slate-700',
  'border-slate-200': 'border-slate-200 dark:border-slate-700',
};

export default replacements;
