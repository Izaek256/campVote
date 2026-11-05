import { redirect } from "next/navigation";

/** The product starts at the signed-in elections list; /login is the doorway. */
export default function Home() {
  redirect("/elections");
}
