import { redirect } from "next/navigation";

/** Admin starts at the dashboard; /login is the doorway. */
export default function Home() {
  redirect("/dashboard/account");
}
