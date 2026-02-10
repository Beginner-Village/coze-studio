include "../base.thrift"

namespace go skill

// Skill information
struct SkillInfo {
    1: optional i64 skill_id        (agw.js_conv="str", api.js_conv="true", api.body="skill_id")
    2: optional i64 space_id        (agw.js_conv="str", api.js_conv="true", api.body="space_id")
    3: optional string name         (api.body="name")
    4: optional string description  (api.body="description")           // Short description (injected into system prompt)
    5: optional string prompt       (api.body="prompt")                // Full instructions (with {resource:name|id:xxx} references)
    6: optional string icon_uri     (api.body="icon_uri")
    7: optional i64 creator_id      (agw.js_conv="str", api.js_conv="true", api.body="creator_id")
    8: optional i64 created_at      (api.body="created_at")
    9: optional i64 updated_at      (api.body="updated_at")
}

// Lightweight reference structure for Bot binding
struct SkillReference {
    1: optional i64 skill_id            (agw.js_conv="str", api.js_conv="true", api.body="skill_id")
    2: optional string skill_name       (api.body="skill_name")        // Cached for display
    3: optional string skill_description (api.body="skill_description") // Cached for display
}
