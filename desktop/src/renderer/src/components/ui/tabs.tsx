import * as React from "react"
import * as TabsPrimitive from "@radix-ui/react-tabs"

import { cn } from "@/lib"

const Tabs = TabsPrimitive.Root

const TabsList = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>
>(({ className, children, ...props }, ref) => {
  const innerRef = React.useRef<HTMLDivElement | null>(null)
  const [indicator, setIndicator] = React.useState({ x: 0, w: 0, ready: false })

   
  React.useEffect(() => {
    const el = innerRef.current
    if (!el) return

    const update = () => {
      const active = el.querySelector<HTMLElement>('[data-state="active"]')
      if (active) {
        setIndicator({ x: active.offsetLeft, w: active.offsetWidth, ready: true })
      }
    }

    update()

    const mo = new MutationObserver(update)
    mo.observe(el, { attributes: true, subtree: true, attributeFilter: ["data-state"] })

    const ro = new ResizeObserver(update)
    ro.observe(el)

    return () => {
      mo.disconnect()
      ro.disconnect()
    }
  }, [])

  return (
    <TabsPrimitive.List
      ref={(el) => {
        innerRef.current = el
        if (typeof ref === "function") ref(el)
        else if (ref) (ref as React.MutableRefObject<typeof el>).current = el
      }}
      className={cn(
        "relative inline-flex h-10 items-center gap-1 border-b border-border text-muted-foreground",
        className
      )}
      {...props}
    >
      {children}
      <span
        className="absolute bottom-0 h-0.5 rounded-full bg-primary transition-all duration-300 ease-[cubic-bezier(0.4,0,0.2,1)]"
        style={{
          transform: `translateX(${indicator.x}px)`,
          width: indicator.w,
          opacity: indicator.ready ? 1 : 0,
        }}
        aria-hidden="true"
      />
    </TabsPrimitive.List>
  )
})
TabsList.displayName = TabsPrimitive.List.displayName

const TabsTrigger = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    className={cn(
      "inline-flex items-center justify-center whitespace-nowrap px-4 pb-3 pt-2 text-sm font-medium ring-offset-background transition-colors duration-200 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:text-foreground",
      className
    )}
    {...props}
  />
))
TabsTrigger.displayName = TabsPrimitive.Trigger.displayName

const TabsContent = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.Content
    ref={ref}
    className={cn(
      "mt-2 animate-fade-in ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
      className
    )}
    {...props}
  />
))
TabsContent.displayName = TabsPrimitive.Content.displayName

export { Tabs, TabsList, TabsTrigger, TabsContent }
