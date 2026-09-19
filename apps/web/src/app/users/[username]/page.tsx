import { notFound } from "next/navigation";
import { Feed } from "@/components/feed";
import { ProfileHeader } from "@/components/profile-header";
import { getPostsByUsername, getUser } from "@/data/mock";

export default async function ProfilePage({
  params,
}: PageProps<"/users/[username]">) {
  const { username } = await params;
  const user = getUser(username);

  if (!user) {
    notFound();
  }

  const items = getPostsByUsername(user.username).map((post) => ({
    post,
    author: user,
  }));

  return (
    <main className="flex-1">
      <ProfileHeader user={user} />
      <Feed items={items} />
    </main>
  );
}
