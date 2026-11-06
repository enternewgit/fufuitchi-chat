// 管理者ログイン
import LoginForm from "@/components/admin/LoginForm";

export default function AdminLoginPage() {
  return (
    <main className="mx-auto w-full max-w-sm px-4 py-10">
      <h1 className="text-xl font-bold">管理者ログイン</h1>
      <LoginForm />
    </main>
  );
}
