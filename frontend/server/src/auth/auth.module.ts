import { Global, Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { User } from 'entities/User.entity';
import { Client } from 'entities/Client.entity';
import { UserRole } from 'entities/UserRole.entity';
import { UserRefreshToken } from 'entities/UserRefreshToken.entity';
import { MailModule } from 'src/mail/mail.module';
import { AuthController } from './controllers/auth.controller';
import { AuthService } from './services/auth.service';
import { AuthThirdPartyController } from './controllers/auth-third-party.controller';
import { ClientAuthController } from './controllers/client-auth.controller';
import { ClientAuthService } from './services/client-auth.service';
import { AuthThirdPartyService } from './services/auth-third-party.service';
import { PostModule } from '../post/post.module';
import { PostFolder } from '../../entities/PostFolder.entity';
import { MembershipModule } from '../membership/membership.module';
import { AuthApiController } from './controllers/auth-api.controller';
import { AuthMetamaskService } from './services/auth-metamask.service';
import { AuthMetamaskController } from './controllers/auth-metamask.controller';

@Global()
@Module({
  controllers: [
    AuthController,
    AuthThirdPartyController,
    ClientAuthController,
    AuthApiController,
    AuthMetamaskController,
  ],
  imports: [
    TypeOrmModule.forFeature([
      User,
      Client,
      UserRefreshToken,
      UserRole,
      PostFolder,
    ]),
    MembershipModule,
    MailModule,
    PostModule,
  ],
  exports: [AuthService],
  providers: [
    AuthService,
    ClientAuthService,
    AuthThirdPartyService,
    AuthMetamaskService,
  ],
})
export class AuthModule {}
