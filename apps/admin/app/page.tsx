import { redirect } from "next/navigation";

/** Admin starts at the account page; /login is the doorway. */
export default function Home() {
  redirect("/account");
}
