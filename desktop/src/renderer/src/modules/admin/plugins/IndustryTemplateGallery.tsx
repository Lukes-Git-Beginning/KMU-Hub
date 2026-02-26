/**
 * Grid of industry template cards with "Apply" button.
 *
 * Shows available industry templates that can be applied to configure
 * plugins with industry-specific defaults, validation rules, and
 * workflow rules.
 */
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { useState } from 'react'
import {
  Loader2,
  Package,
  CheckCircle,
  Factory,
  Truck,
  Wrench,
  ShoppingCart,
  Building2,
  Stethoscope,
  GraduationCap,
  Utensils,
} from 'lucide-react'
import { useIndustryTemplates, useApplyTemplate } from '@/api/hooks/usePlugins'
import type { IndustryTemplate } from '@/api/plugin-types'

/** Map icon string to Lucide component */
function templateIcon(icon: string) {
  const iconMap: Record<string, React.ReactNode> = {
    factory: <Factory className="h-6 w-6" />,
    truck: <Truck className="h-6 w-6" />,
    wrench: <Wrench className="h-6 w-6" />,
    'shopping-cart': <ShoppingCart className="h-6 w-6" />,
    building: <Building2 className="h-6 w-6" />,
    stethoscope: <Stethoscope className="h-6 w-6" />,
    graduation: <GraduationCap className="h-6 w-6" />,
    utensils: <Utensils className="h-6 w-6" />,
  }
  return iconMap[icon] ?? <Package className="h-6 w-6" />
}

export function IndustryTemplateGallery() {
  const { data: templates, isLoading } = useIndustryTemplates()
  const applyTemplate = useApplyTemplate()

  const [confirmTemplate, setConfirmTemplate] =
    useState<IndustryTemplate | null>(null)

  const handleApply = async () => {
    if (!confirmTemplate) return
    await applyTemplate.mutateAsync({ template_id: confirmTemplate.id })
    setConfirmTemplate(null)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!templates || templates.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4">
        Keine Branchenvorlagen verfuegbar.
      </p>
    )
  }

  return (
    <>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {templates.map((template) => (
          <Card key={template.id} className="flex flex-col">
            <CardHeader className="pb-3">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  {templateIcon(template.icon)}
                </div>
                <div className="min-w-0 flex-1">
                  <CardTitle className="text-sm font-medium truncate">
                    {template.name}
                  </CardTitle>
                  <CardDescription className="text-xs">
                    {template.industry}
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent className="flex-1 pb-3">
              <p className="text-sm text-muted-foreground line-clamp-2">
                {template.description}
              </p>
              <div className="flex flex-wrap gap-1.5 mt-3">
                {template.custom_fields.length > 0 && (
                  <Badge variant="outline" className="text-xs">
                    {template.custom_fields.length} Felder
                  </Badge>
                )}
                {template.validation_rules.length > 0 && (
                  <Badge variant="outline" className="text-xs">
                    {template.validation_rules.length} Validierungen
                  </Badge>
                )}
                {template.workflow_rules.length > 0 && (
                  <Badge variant="outline" className="text-xs">
                    {template.workflow_rules.length} Workflows
                  </Badge>
                )}
              </div>
            </CardContent>
            <CardFooter className="pt-0">
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                onClick={() => setConfirmTemplate(template)}
              >
                Vorlage anwenden
              </Button>
            </CardFooter>
          </Card>
        ))}
      </div>

      {/* Confirmation dialog */}
      <Dialog
        open={!!confirmTemplate}
        onOpenChange={(open) => !open && setConfirmTemplate(null)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Branchenvorlage anwenden</DialogTitle>
            <DialogDescription>
              Moechten Sie die Vorlage &quot;{confirmTemplate?.name}&quot;
              anwenden? Dadurch werden vorkonfigurierte Felder,
              Validierungsregeln und Workflows erstellt.
            </DialogDescription>
          </DialogHeader>

          {confirmTemplate && (
            <div className="space-y-2 text-sm">
              <div className="flex items-center gap-2">
                <CheckCircle className="h-3.5 w-3.5 text-green-500" />
                <span>
                  {confirmTemplate.custom_fields.length} benutzerdefinierte
                  Felder
                </span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle className="h-3.5 w-3.5 text-green-500" />
                <span>
                  {confirmTemplate.validation_rules.length} Validierungsregeln
                </span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle className="h-3.5 w-3.5 text-green-500" />
                <span>
                  {confirmTemplate.workflow_rules.length} Workflow-Regeln
                </span>
              </div>
            </div>
          )}

          <DialogFooter>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setConfirmTemplate(null)}
            >
              Abbrechen
            </Button>
            <Button
              size="sm"
              onClick={handleApply}
              disabled={applyTemplate.isPending}
            >
              {applyTemplate.isPending ? (
                <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
              ) : (
                <Package className="h-3.5 w-3.5 mr-1.5" />
              )}
              Anwenden
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
