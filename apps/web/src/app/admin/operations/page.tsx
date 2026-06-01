import { OperationsDashboard } from "@/features/operations/operations-dashboard";
import { WebhookHealthPanel } from "@/features/operations/webhook-health-panel";

export default function OperationsPage() {
  return (
    <div className="flex flex-col gap-6">
      <OperationsDashboard />
      <WebhookHealthPanel />
    </div>
  );
}
