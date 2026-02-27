import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib"

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        secondary:
          "border border-border/50 bg-secondary text-secondary-foreground hover:bg-secondary/80",
        destructive:
          "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
        outline: "text-foreground",
        accent:
          "border-transparent bg-[var(--accent-1)] text-white hover:opacity-90",
        gradient:
          "border-transparent bg-gradient-to-r from-[var(--accent-1)] to-[var(--accent-2)] text-white",
        dot: "border border-border/50 bg-secondary text-secondary-foreground pl-2",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {
  dotColor?: string
}

function Badge({ className, variant, dotColor, children, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props}>
      {variant === "dot" && (
        <span
          className="mr-1.5 h-1.5 w-1.5 rounded-full"
          style={{ backgroundColor: dotColor || "var(--primary)" }}
        />
      )}
      {children}
    </div>
  )
}

export { Badge, badgeVariants }
