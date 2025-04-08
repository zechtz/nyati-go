import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-primary-500 text-primary-foreground hover:bg-primary-600",
        secondary:
          "border-transparent bg-gray-500 text-gray-foreground hover:bg-gray-600",
        success:
          "border-transparent bg-green-500 text-green-foreground hover:bg-green-600",
        warning:
          "border-transparent bg-yellow-500 text-yellow-foreground hover:bg-yellow-600",
        destructive:
          "border-transparent bg-red-500 text-red-foreground hover:bg-red-600",
        outline: "text-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
