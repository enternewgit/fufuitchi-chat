// キャラの一覧・追加・編集
import AdminNav from "@/components/admin/AdminNav";
import CharacterEditor from "@/components/admin/CharacterEditor";

export default function AdminCharactersPage() {
  return (
    <main className="mx-auto w-full max-w-5xl px-4 py-6">
      <AdminNav />
      <CharacterEditor />
    </main>
  );
}
