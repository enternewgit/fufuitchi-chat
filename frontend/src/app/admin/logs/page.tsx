// 会話ログの一覧・詳細
import AdminNav from "@/components/admin/AdminNav";
import LogViewer from "@/components/admin/LogViewer";

export default function AdminLogsPage() {
  return (
    <main className="mx-auto w-full max-w-5xl px-4 py-6">
      <AdminNav />
      <LogViewer />
    </main>
  );
}
