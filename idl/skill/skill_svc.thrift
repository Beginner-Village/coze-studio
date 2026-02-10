include "../base.thrift"
include "skill.thrift"

namespace go skill

// Create Skill
struct CreateSkillRequest {
    1: required i64 space_id        (agw.js_conv="str", api.js_conv="true", api.body="space_id")
    2: required string name         (api.body="name")
    3: optional string description  (api.body="description")
    4: optional string prompt       (api.body="prompt")
    5: optional string icon_uri     (api.body="icon_uri")

    255: base.Base Base (api.none="true")
}

struct CreateSkillData {
    1: skill.SkillInfo skill_info
}

struct CreateSkillResponse {
    253: required i32 code
    254: required string msg
    1: CreateSkillData data
    255: required base.BaseResp BaseResp (api.none="true")
}

// Get Skill
struct GetSkillRequest {
    1: required i64 skill_id        (agw.js_conv="str", api.js_conv="true", api.query="skill_id")
    2: required i64 space_id        (agw.js_conv="str", api.js_conv="true", api.query="space_id")

    255: base.Base Base (api.none="true")
}

struct GetSkillData {
    1: skill.SkillInfo skill_info
}

struct GetSkillResponse {
    253: required i32 code
    254: required string msg
    1: GetSkillData data
    255: required base.BaseResp BaseResp (api.none="true")
}

// Update Skill
struct UpdateSkillRequest {
    1: required i64 skill_id        (agw.js_conv="str", api.js_conv="true", api.body="skill_id")
    2: required i64 space_id        (agw.js_conv="str", api.js_conv="true", api.body="space_id")
    3: optional string name         (api.body="name")
    4: optional string description  (api.body="description")
    5: optional string prompt       (api.body="prompt")
    6: optional string icon_uri     (api.body="icon_uri")

    255: base.Base Base (api.none="true")
}

struct UpdateSkillData {
    1: skill.SkillInfo skill_info
}

struct UpdateSkillResponse {
    253: required i32 code
    254: required string msg
    1: UpdateSkillData data
    255: required base.BaseResp BaseResp (api.none="true")
}

// Delete Skill
struct DeleteSkillRequest {
    1: required i64 skill_id        (agw.js_conv="str", api.js_conv="true", api.body="skill_id")
    2: required i64 space_id        (agw.js_conv="str", api.js_conv="true", api.body="space_id")

    255: base.Base Base (api.none="true")
}

struct DeleteSkillResponse {
    253: required i32 code
    254: required string msg
    255: required base.BaseResp BaseResp (api.none="true")
}

// List Skills
struct ListSkillsRequest {
    1: required i64 space_id        (agw.js_conv="str", api.js_conv="true", api.query="space_id")
    2: optional i32 page            (api.query="page")
    3: optional i32 page_size       (api.query="page_size")
    4: optional string keyword      (api.query="keyword")

    255: base.Base Base (api.none="true")
}

struct ListSkillsData {
    1: list<skill.SkillInfo> skill_list
    2: i32 total
}

struct ListSkillsResponse {
    253: required i32 code
    254: required string msg
    1: ListSkillsData data
    255: required base.BaseResp BaseResp (api.none="true")
}

// Skill management service
service SkillService {
    CreateSkillResponse CreateSkill(1: CreateSkillRequest req) (api.post="/api/skill/create")
    GetSkillResponse GetSkill(1: GetSkillRequest req) (api.get="/api/skill/get")
    UpdateSkillResponse UpdateSkill(1: UpdateSkillRequest req) (api.post="/api/skill/update")
    DeleteSkillResponse DeleteSkill(1: DeleteSkillRequest req) (api.post="/api/skill/delete")
    ListSkillsResponse ListSkills(1: ListSkillsRequest req) (api.get="/api/skill/list")
}
