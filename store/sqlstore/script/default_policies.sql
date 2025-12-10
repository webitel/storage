with available_domains as (
    select wd.dc
    from directory.wbt_domain wd
)
insert into storage.file_policies(
        domain_id,
        created_at,
        updated_at,
        created_by,
        updated_by,
        name,
        enabled,
        mime_types,
        speed_download,
        speed_upload,
        channels,
        retention_days,
        max_upload_size
    )
select ad.dc,
    now(),
    now(),
    null,
    null,
    p.name,
    false,
    p.mime_types,
    p.speed_download,
    p.speed_upload,
    p.channels,
    p.retention_days,
    p.max_upload_size
from available_domains ad
    cross join (
        values (
                'media_storage',
                array ['image/jpeg', 'image/png', 'video/mp4', 'audio/mpeg', 'audio/wav'],
                2048,
                1024,
                array ['chat'],
                365,
                52428800
            ),
            (
                'email_attachment',
                array ['application/pdf', 'application/msword', 
				'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'image/*'],
                1024,
                512,
                array ['email'],
                365,
                10485760
            ),
            (
                'call_recordings',
                array ['audio/mpeg', 'audio/wav'],
                1024,
                512,
                array ['call'],
                365,
                20971520
            )
    ) as p(
        name,
        mime_types,
        speed_download,
        speed_upload,
        channels,
        retention_days,
        max_upload_size
    )
where not exists (
        select 1
        from storage.file_policies fp
        where fp.domain_id = ad.dc
            and fp.name = p.name
    );