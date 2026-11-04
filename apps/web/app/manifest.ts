import type { MetadataRoute } from "next";
import { CAMPUS_NAME } from "@/lib/config";

/**
 * Web app manifest. The icon set is the Option D ballot-box mark: an ink square
 * with a white ballot box, drawn as an SVG so one file serves every size.
 */
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: `campVote — ${CAMPUS_NAME} elections`,
    short_name: "campVote",
    description: `Vote in authorised ${CAMPUS_NAME} elections.`,
    start_url: "/",
    display: "standalone",
    background_color: "#FFFFFF",
    theme_color: "#12783F",
    icons: [{ src: "/icon.svg", sizes: "any", type: "image/svg+xml" }],
  };
}
